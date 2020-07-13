// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"sigs.k8s.io/yaml"

)

var (
	PrometheusConfigEmptyFile                   = errors.New("Error: Invalid Prometheus YAML: it is empty. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigScrapeIntervalNotDefined    = errors.New("Error: Invalid Prometheus YAML: it does not have the mandatory name global.scrape_interval. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigRuleFilesNotDefined         = errors.New("Error: Prometheus YAML does not have rule_files defined. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigJobsNotDefined              = errors.New("Error: Prometheus YAML does not have scrape_configs jobs defined. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigJobPrometheusNotDefined     = errors.New("Error: Prometheus YAML does not have job scrape_configs[0].job_name=prometheus defined. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigJobPushGatewayNotDefined    = errors.New("Error: Prometheus YAML does not have job scrape_configs[1].job_name=PushGateway defined. Please do a get of the existing Prometheus config file and append to it.")
	PrometheusConfigJobKubernetesPodsNotDefined = errors.New("Error: Prometheus YAML does not have job scrape_configs[2].job_name=kubernetes-pods defined. Please do a get of the existing Prometheus config file and append to it.")
)

func (k *K8s) GetPrometheusConfig(w http.ResponseWriter, r *http.Request) {

	mapPath := PrometheusConfigMapPath
	configName := "prometheus-config"
	keyName := PrometheusConfigFileName

	// Was a timestamp provided?
	var version = ""
	if len(r.FormValue("version")) > 0 {
		version = r.FormValue("version")

		// Validate the timestamp
		re, _ := regexp.Compile("^[0-9-T]*$")
		actVersion := re.ReplaceAllString(version, "")
		if actVersion != "" {
			badRequest(w, "ERROR: The version timestamp provided is not valid.")
			return
		}

		mapPath = PrometheusVersionsConfigMapPath
		configName = configName + "-versions"
		keyName = keyName + "-" + version
	}

	// Get the proper configMap
	_, configMap, err := k.getConfigMapByPath(mapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read %s ConfigMap: %v", configName, err))
		return
	}

	// Respond appropriately if the configmap is empty.
	if len(configMap) == 0 {
		if configName == "prometheus-config" {
			internalError(w, "The "+configName+" configMap appears to be empty.")
		} else {
			notFoundError(w, "No older versions of the Prometheus configuration were found.")
		}
		return
	}

	// Results were returned, look for the requested key
	configValue := configMap[keyName]
	if len(configValue) == 0 {
		notFoundError(w, "Unable to find the requested Prometheus configuration.")
		return
	}
	success(w, configValue)
}

func (k *K8s) GetPrometheusVersions(w http.ResponseWriter, r *http.Request) {

	var result []byte
	resultMap := make(map[string][]string)
	resultMap["versions"] = make([]string, 0)

	// Get the prometheus-config-versions ConfigMap
	_, configMap, err := k.getConfigMapByPath(PrometheusVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read prometheus-config-versions ConfigMap: %v", err))
		return
	}

	// Go capture the keys from versions configMap. We need to sort & JSON-ify the output
	// We only want to return the timestamps
	keyList := k.sortKeysFromConfigMap(configMap, PrometheusConfigFileName)

	// If no keys are found, then we have no saved configs
	if len(keyList) == 0 {
		success(w, "No older versions of the Prometheus configuration were found.")
		return
	}
	for k := range keyList {
		resultMap["versions"] = append(resultMap["versions"], strings.Replace(keyList[k], PrometheusConfigFileName+"-", "", 1))
	}
	result, _ = json.MarshalIndent(resultMap, "", "\t")
	successBytes(w, result)

}

func (k *K8s) PutPrometheusConfig(w http.ResponseWriter, r *http.Request) {
	b, e := ioutil.ReadAll(r.Body)
	if e != nil {
		internalError(w, "Unable to read request Body: "+e.Error())
		return
	}

	// Convert provided update in the body to json and parse
	jsonObject, e := yaml.YAMLToJSON(b)
	if e != nil {
		badRequest(w, "Unable to convert the provided YAML to JSON: "+e.Error())
		return
	}
	jsonParsedObj, e := gabs.ParseJSON([]byte(string(jsonObject)))
	if e != nil {
		internalError(w, "Unable to parse JSON: "+e.Error())
		return
	}

	// Validate this is a proper prometheus yaml. i.e. customers have not removed stuff added by VMI Team.
	if validStatus, e := ValidateVMIPrometheusElements(jsonParsedObj); e != nil || !validStatus {
		badRequest(w, "Prometheus configuration was not updated. Reserved section of prometheus.yml was altered: "+e.Error())
		return
	}

	// Validate with promtool
	promOut, e := checkPrometheusConfig(b)
	log(LevelInfo, "%s\n", promOut)
	if e != nil {
		badRequest(w, "Prometheus configuration was not updated.  Failed to validate with promtool: "+string(promOut)+" :ErrorMsg: "+e.Error())
		return
	}

	// Get the configmaps
	currentConfigMapName, currentConfigMap, err := k.getConfigMapByPath(PrometheusConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read prometheus-config ConfigMap: %v", err))
		return
	}
	savedConfigMapName, savedConfigMap, err := k.getConfigMapByPath(PrometheusVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read prometheus-config-versions ConfigMap: %v", err))
		return
	}

	// Special check... did the user make any changes?  If not, take no action and exit
	if currentConfigMap[PrometheusConfigFileName] == string(b) {
		success(w, "The provided body is identical to the current Prometheus configuration. No action will be taken.")
		return
	}

	// Copy the current prometheus.yml to the versions ConfigMap
	// Default name all keys is:  "prometheus.yml-TIMESTAMP"
	timeNow := time.Now().UTC()
	keyName := PrometheusConfigFileName + "-" + timeNow.Format(Layout)

	// One-time step:  need to initialize the empty map the first time
	if savedConfigMap == nil {
		savedConfigMap = make(map[string]string)
	}
	savedConfigMap[keyName] = currentConfigMap[PrometheusConfigFileName]

	// How many backups do we have?  Do we need to delete any old ones?
	// K8S configmaps have limited space; very large configs can fill the versions configMap.
	keyList := k.sortKeysFromConfigMap(savedConfigMap, PrometheusConfigFileName)
	if len(keyList) > MaxBackupFiles {
		for j := range keyList {

			// We want to keep at least MaxBackupFiles...
			if j < MaxBackupFiles {
				continue
			}
                        // Delete any backups that are MaxBackupHours or older.
                        if k.isOldVersion(keyList[j], PrometheusConfigFileName, timeNow) {
                                delete(savedConfigMap, keyList[j])
                        }
		}
	}

	// Okay, updating the older configMap is completed.  Save it!
	e = k.updateConfigMapByName(savedConfigMap, savedConfigMapName)
	if e != nil {
		internalError(w, "Unable to save a backup of prometheus.yml to prometheus-config-versions ConfigMap. "+e.Error())
		return
	}

	// Finally, update the current Configmap with the new version (validated) provided by the user
	currentConfigMap[PrometheusConfigFileName] = string(b)
	e = k.updateConfigMapByName(currentConfigMap, currentConfigMapName)
	if e != nil {
		internalError(w, "Unable to update prometheus-config ConfigMap: "+e.Error())
		return
	}
	// returning HTTP status "202: Accepted".
	// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
	// before the response is sent.
	accepted(w, "The Prometheus configuration is being updated.")
}

// "Validate that the Prometheus config contains all the stuff VMI team added"
func ValidateVMIPrometheusElements(g *gabs.Container) (bool, error) {
	var scrapeConfig = "scrape_configs"
	var jobNameParameter = "job_name"
	var configString = g.String()
	log(LevelInfo, "%s", "Prometheus Config: "+ g.String() )

	if  configString == "null" {
		return false, PrometheusConfigEmptyFile
	}
	if !g.ExistsP("global.scrape_interval") {
		return false, PrometheusConfigScrapeIntervalNotDefined
	}
	if !g.ExistsP("rule_files") {
		return false, PrometheusConfigRuleFilesNotDefined
	}

	noOfScrapeConfigs, e := g.ArrayCountP(scrapeConfig)
	if e != nil {
		log(LevelError, "%s", "Unable to read scrape configs: "+e.Error())
		return false, e
	}
	if noOfScrapeConfigs < 3 {
		return false, PrometheusConfigJobsNotDefined
	}

	jobNameElement, e := g.ArrayElementP(0, scrapeConfig)
	jobName := jobNameElement.Path(jobNameParameter).Data().(string)
	if e != nil || jobName == "" {
		log(LevelError, "%s", "Unable to read scrape config job_name: prometheus: "+e.Error())
		return false, e
	}

	if jobName != "prometheus" {
		return false, PrometheusConfigJobPrometheusNotDefined
	}

	jobNameElement, e = g.ArrayElementP(1, scrapeConfig)
	jobName = jobNameElement.Path(jobNameParameter).Data().(string)
	if e != nil || jobName == "" {
		log(LevelError, "%s", "Unable to read scrape config job_name: PushGateway: "+e.Error())
		return false, e
	}

	if jobName != "PushGateway" {
		return false, PrometheusConfigJobPushGatewayNotDefined
	}

	jobNameElement, e = g.ArrayElementP(2, scrapeConfig)
	jobName = jobNameElement.Path(jobNameParameter).Data().(string)
	if e != nil || jobName == "" {
		log(LevelError, "%s", "Unable to read scrape config job_name: kubernetes-pods: "+e.Error())
		return false, e
	}

	if jobName != "kubernetes-pods" {
		return false, PrometheusConfigJobKubernetesPodsNotDefined
	}

	return true, nil
}

func checkPrometheusConfig(b []byte) ([]byte, error) {

	tf, e := saveDataToTempFile(b)
	if e != nil {
		log(LevelError, "failed to create temp file: %v \n", e)
		return nil, e
	}
	defer os.Remove(tf.Name())

	promtoolCommand := execute(promtoolPath, "check", "config", tf.Name())
	promtoolOutput, err := promtoolCommand.CombinedOutput()
	if err != nil {
		log(LevelDebug, "%s check config %s failed: (%s) %v\n",
			promtoolPath, tf.Name(), promtoolOutput, err)
	}
	return promtoolOutput, err
}
