// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"sigs.k8s.io/yaml"
)

// Use static filename for global rules to make PUT idempotent.
const unnamedRules = "unnamed.rules"

var (
	PrometheusRuleEmptyFile                     = errors.New("Error: Invalid Rule, It is empty")
	PrometheusRuleRootElementGroupsNotDefined   = errors.New("Error: Invalid Rule, does not have root element groups: defined")
	PrometheusRuleDoesNotHaveSingleGroupDefined = errors.New("Error: Invalid Rule, Does not have even a single rule group defined")
)

// Return a JSON with names of all current Prometheus rules files
func (k *K8s) GetPrometheusRuleNames(w http.ResponseWriter, r *http.Request) {

	var result []byte
	resultMap := make(map[string][]string)
	resultMap["alertrules"] = make([]string, 0)

	// Get the current alertrules configmap
	_, configMap, err := k.getConfigMapByPath(PrometheusRulesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read the alertrules ConfigMap: %v", err))
		return
	}
	if len(configMap) == 0 {
		success(w, "No Prometheus alert rules were found.")
		return
	}

	// Grab all the keys from the configmap and return a sorted list.
	ruleNames := make([]string, len(configMap))
	i := 0
	for k := range configMap {
		ruleNames[i] = k
		i++
	}
	sort.Strings(ruleNames)
	for k := range ruleNames {
		resultMap["alertrules"] = append(resultMap["alertrules"], ruleNames[k])
	}
	result, _ = json.MarshalIndent(resultMap, "", "\t")
	successBytes(w, result)
}

// Take the user-provided rule file name and return a JSON with all available
// versions of that file.
func (k *K8s) GetPrometheusRuleVersions(w http.ResponseWriter, r *http.Request) {

	var result []byte
	resultMap := make(map[string][]string)
	resultMap["versions"] = make([]string, 0)

	fileName := path.Base(path.Dir(r.URL.Path))

	// Validate the file name provided
	if !strings.HasSuffix(fileName, ".rules") {
		badRequest(w, "ERROR: The file name provided must end with: .rules")
		return
	}
	err := validateName(fileName)
	if err != nil {
		badRequest(w, "ERROR: Invalid File Name: "+err.Error())
		return
	}

	// Go check that the user requested a real rules file
	_, currentConfigMap, err := k.getConfigMapByPath(PrometheusRulesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules ConfigMap: %v", err))
		return
	}
	exists := false
	for k := range currentConfigMap {
		if k == fileName {
			exists = true
		}
	}
	if !exists {
		notFoundError(w, "ERROR: Unable to find a current Prometheus Alert rule called: "+fileName)
		return
	}

	// Get the saved configmap
	_, savedConfigMap, err := k.getConfigMapByPath(PrometheusRulesVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules-versions ConfigMap: %v", err))
		return
	}
	// Exit immediately if there are no saved rules - this is not an error
	if len(savedConfigMap) == 0 {
		success(w, "No older versions of the Prometheus alert rule were found.")
		return
	}

	// Search the saved configmap for any matching alert rules
	// If any keys are found, stuff the timestamp in the resultMap map
	keyList := k.sortKeysFromConfigMap(savedConfigMap, fileName)
	if len(keyList) != 0 {
		for k := range keyList {
			resultMap["versions"] = append(resultMap["versions"], strings.Replace(keyList[k], fileName+"-", "", 1))
		}
	}

	// Return a friendly message if no older versions were found.
	if len(resultMap["versions"]) == 0 {
		success(w, "No older versions of the Prometheus alert rule were found.")
		return
	}

	// Return a JSON of the timestamps found
	result, _ = json.MarshalIndent(resultMap, "", "\t")
	successBytes(w, result)
}

// Return the contents of the requested Alert Rules file
// All rules files must end in suffix ".rules" so we can differentiate requests for current vs. saved versions.
func (k *K8s) GetPrometheusRules(w http.ResponseWriter, r *http.Request) {

	configName := "alertrules"

	// Validate the file
	fileName := path.Base(r.URL.Path)
	if fileName == "" {
		badRequest(w, "ERROR:  No file name was provided.")
		return
	}
	if !strings.Contains(fileName, ".rules") {
		badRequest(w, "ERROR:  The requested file does not appear to be a valid file name.")
		return
	}
	err := validateName(fileName)
	if err != nil {
		badRequest(w, "ERROR: Invalid File Name: "+err.Error())
		return
	}

	// Validate the timestamp
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
	}

	// Go check that the user requested a real rules file
	_, configMap, err := k.getConfigMapByPath(PrometheusRulesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules ConfigMap: %v", err))
		return
	}
	exists := false
	for k := range configMap {
		if k == fileName {
			exists = true
		}
	}
	if !exists {
		notFoundError(w, "Unable to find a current Prometheus Alert rule called: "+fileName)
		return
	}

	// Was a timestamp provided?
	// This means the user wants the contents of an older saved version
	if version != "" {
		_, configMap, err = k.getConfigMapByPath(PrometheusRulesVersionsConfigMapPath)
		if err != nil {
			internalError(w, fmt.Sprintf("Unable to read alertrules-versions ConfigMap: %v", err))
			return
		}
		configName = "alertrules-versions"
		fileName = fileName + "-" + version
	}

	// Go get the requested file
	for k, v := range configMap {
		if k == fileName {
			log(LevelDebug, "%s", "Found requested file: "+fileName+" in "+configName+" configMap")
			success(w, v)
			return
		}
	}

	// If not found, return an appropriate error...
	error := ""
	if version != "" {
		error = "Unable to find the requested file version: " + version
	} else {
		error = "Unable to find the requested alert rules file: " + fileName
	}
	notFoundError(w, error)
}

// Removed the requested current Alert Rules file.
// If the optional flag is included, delete all the rules saved backups too.
func (k *K8s) DeletePrometheusRules(w http.ResponseWriter, r *http.Request) {

	// Validate the file
	fileName := path.Base(r.URL.Path)
	if fileName == "" {
		badRequest(w, "ERROR:  No file name was provided.")
		return
	}
	if !strings.HasSuffix(fileName, ".rules") {
		badRequest(w, "ERROR: File name must end with: .rules")
		return
	}
	err := validateName(fileName)
	if err != nil {
		badRequest(w, "ERROR: Unable to validate the provided file name.")
		return
	}

	// Go get the configmaps
	currentConfigMapName, currentConfigMap, err := k.getConfigMapByPath(PrometheusRulesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules ConfigMap: %v", err))
		return
	}
	savedConfigMapName, savedConfigMap, err := k.getConfigMapByPath(PrometheusRulesVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules-versions ConfigMap: %v", err))
		return
	}

	// One-time step:  need to initialize the empty map the first time
	if savedConfigMap == nil {
		savedConfigMap = make(map[string]string)
	}

	// Go find the file we want to delete
	for j := range currentConfigMap {
		if j == fileName {
			log(LevelDebug, "%s", "Found existing file: "+fileName+" in alertrules configMap")

			// Delete the current version.
			delete(currentConfigMap, j)
			e := k.updateConfigMapByName(currentConfigMap, currentConfigMapName)
			if e != nil {
				internalError(w, "Unable to update alertrules ConfigMap. "+e.Error())
				return
			}

			// Delete all the saved versions too
			keyList := k.sortKeysFromConfigMap(savedConfigMap, fileName)
			for s := range keyList {
				delete(savedConfigMap, keyList[s])
			}
			e = k.updateConfigMapByName(savedConfigMap, savedConfigMapName)
			if e != nil {
				internalError(w, "Unable to update alertrules-versions ConfigMap. "+e.Error())
				return
			}

			// returning HTTP status "202: Accepted".
			// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
			// before the response is sent.
			accepted(w, "The current alert rule: "+fileName+" and all older versions are being deleted.")
			return
		}
	}
	notFoundError(w, "No action taken. Unable to find a current alert rule called: "+fileName)
}

// PUT /prometheus/rules has been deprecated.  Return a friendly error message instead.
func (k *K8s) PutPrometheusUnnamedRules(w http.ResponseWriter, r *http.Request) {
	badRequest(w, fmt.Sprintf("ERROR:  This endpoint has been deprecated in Cirith v1.\nPlease use:  PUT /prometheus/rules/%s", unnamedRules))
}

// Update the requested Alert Rules file with the provided body.
// If not a new rule, save a backup copy of the current rule.
func (k *K8s) PutPrometheusRules(w http.ResponseWriter, r *http.Request) {
	b, e := ioutil.ReadAll(r.Body)
	if e != nil {
		internalError(w, "ERROR: Unable to read request Body.")
		return
	}

	// Validate the provided file name
	fileName := path.Base(r.URL.Path)
	if fileName == "" {
		badRequest(w, "ERROR:  No file name was provided.")
		return
	}
	if fileName == ".rules" {
		badRequest(w, "ERROR: Invalid file name.")
		return
	}
	if !strings.HasSuffix(fileName, ".rules") {
		badRequest(w, "ERROR: File name must end with: .rules")
		return
	}
	e = validateName(fileName)
	if e != nil {
		badRequest(w, "ERROR: The file name provided is invalid.")
		return
	}

	// Convert provided update in the body to json and parse
	jsonObject, e := yaml.YAMLToJSON(b)
	if e != nil {
		badRequest(w, "ERROR: Unable to convert YAML to JSON.")
		return
	}
	jsonParsedObj, e := gabs.ParseJSON([]byte(string(jsonObject)))
	if e != nil {
		internalError(w, "Unable to parse JSON")
		return
	}

	// Validate this is a proper Rule file.
	if validStatus, e := ValidatePrometheusRuleElements(jsonParsedObj); e != nil || !validStatus {
		badRequest(w, "No action taken. Did not create/update Rule: "+e.Error())
		return
	}

	// Validate with promtool
	promOut, e := checkPrometheusRules(b)
	log(LevelDebug, "%s\n", promOut)
	if e != nil {
		badRequest(w, "No action taken.  Failed to validate with promtool: "+string(promOut)+" :ErrorMsg: "+e.Error())
		return
	}

	// Go get the configmaps
	currentConfigMapName, currentConfigMap, err := k.getConfigMapByPath(PrometheusRulesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules ConfigMap: %v", err))
		return
	}
	savedConfigMapName, savedConfigMap, err := k.getConfigMapByPath(PrometheusRulesVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertrules-versions ConfigMap: %v", err))
		return
	}

	// One-time step:  need to initialize the empty map the first time
	if savedConfigMap == nil {
		savedConfigMap = make(map[string]string)
	}

	// Does this rule already exist?
	for j := range currentConfigMap {
		if j == fileName {
			log(LevelDebug, "%s", "Found existing file: "+fileName+" in alertrules configMap")

			// Special check... did the user actually make any updates?  If not, take no action and exit
			if currentConfigMap[fileName] == string(b) {
				success(w, "The provided body is identical to the current Alert Rule: "+fileName+". No action will be taken.")
				return
			}

			// We need to back up the current file first
			timeNow := time.Now().UTC()
			keyName := fileName + "-" + timeNow.Format(Layout)
			savedConfigMap[keyName] = currentConfigMap[fileName]

        		// How many backups do we have?  Do we need to delete any old ones?
        		// K8S configmaps have limited space; very large configs can fill the versions configMap.
        		keyList := k.sortKeysFromConfigMap(savedConfigMap, fileName)
        		if len(keyList) > MaxBackupFiles {
                		for j := range keyList {

                        		// We want to keep at least MaxBackupFiles...
                        		if j < MaxBackupFiles {
                                		continue
                        		}
					// Delete any backups that are MaxBackupHours or older.
					if k.isOldVersion(keyList[j], fileName, timeNow) {
						delete(savedConfigMap, keyList[j])
					}
                		}
        		}

			// Okay, we're done messing with the savedConfigMap... Save it
			e = k.updateConfigMapByName(savedConfigMap, savedConfigMapName)
			if e != nil {
				internalError(w, "Unable to update alertrules-versions ConfigMap. "+e.Error())
				return
			}

			// Finally, update the current configmap
			currentConfigMap[fileName] = string(b)
			e = k.updateConfigMapByName(currentConfigMap, currentConfigMapName)
			if e != nil {
				internalError(w, "Unable to update alertrules ConfigMap. "+e.Error())
				return
			}
			// returning HTTP status "202: Accepted".
			// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
			// before the response is sent.
			accepted(w, "The existing rule: "+fileName+" is being updated.")
			return
		}
	}

	// Apparently, no such rule exists, so we'll create a new one.
	// Need a one-time step to initialize the map if this is the first rule...
	if currentConfigMap == nil {
		currentConfigMap = make(map[string]string)
	}
	currentConfigMap[fileName] = string(b)

	// Update the current configmap
	e = k.updateConfigMapByName(currentConfigMap, currentConfigMapName)
	if e != nil {
		internalError(w, "Unable to update the alertrules configmap: "+e.Error())
		return
	}
	// returning HTTP status "202: Accepted".
	// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
	// before the response is sent.
	accepted(w, "A new rule file: "+fileName+" is being created.")
}

// Do some basic validation on the rule file
func ValidatePrometheusRuleElements(g *gabs.Container) (bool, error) {
	var ruleTopElement = "groups"

	if g.String() == "{}" {
		return false, PrometheusRuleEmptyFile
	}

	if !g.ExistsP(ruleTopElement) {
		return false, PrometheusRuleRootElementGroupsNotDefined
	}

	_, e := g.ArrayCountP(ruleTopElement)
	if e != nil {
		log(LevelError, "%s", "Did not find any rules group: "+e.Error())
		return false, PrometheusRuleDoesNotHaveSingleGroupDefined
	}

	log(LevelDebug, "%s", "Prometheus Rule: "+g.String())
	return true, nil
}

func checkPrometheusRules(b []byte) ([]byte, error) {
	tf, e := saveDataToTempFile(b)
	if e != nil {
		log(LevelError, "failed to create temp file for: (%s) %v\n", tf.Name(), e)
		return nil, e
	}

	defer os.Remove(tf.Name())

	promtoolCommand := execute(promtoolPath, "check", "rules", tf.Name())
	promtoolOutput, err := promtoolCommand.CombinedOutput()
	if err != nil {
		log(LevelDebug, "%s check rules %s failed: (%s) %v\n",
			promtoolPath, tf.Name(), promtoolOutput, err)
	}
	return promtoolOutput, err
}
