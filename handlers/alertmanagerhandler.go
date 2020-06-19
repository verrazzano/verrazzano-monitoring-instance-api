// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func (k *K8s) GetAlertmanagerConfig(w http.ResponseWriter, r *http.Request) {

	mapPath := AlertmanagerConfigMapPath
	configName := "alertmanager-config"
	keyName := AlertmanagerConfigFileName

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

		mapPath = AlertmanagerVersionsConfigMapPath
		configName = configName + "-versions"
		keyName = keyName + "-" + version
	}

	// Get the proper configMap.
	_, configMap, err := k.getConfigMapByPath(mapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read %s ConfigMap: %v", configName, err))
		return
	}

	// Respond appropriately if the configmap is empty.
	if len(configMap) == 0 {
		if configName == "alertmanager-config" {
			internalError(w, "The "+configName+" configMap appears to be empty.")
		} else {
			notFoundError(w, "No older versions of the Alertmanager configuration were found.")
		}
		return
	}

	configValue := configMap[keyName]
	if len(configValue) == 0 {
		notFoundError(w, "Unable to find the requested Alertmanager configuration.")
		return
	}
	success(w, configValue)
}

func (k *K8s) GetAlertmanagerVersions(w http.ResponseWriter, r *http.Request) {

	var result []byte
	resultMap := make(map[string][]string)
	resultMap["versions"] = make([]string, 0)

	// Get the alertmanager-config-versions ConfigMap
	_, configMap, err := k.getConfigMapByPath(AlertmanagerVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertmanager-config-versions ConfigMap: %v", err))
		return
	}

	// We only want to return the timestamps
	keyList := k.sortKeysFromConfigMap(configMap, AlertmanagerConfigFileName)

	// If no keys are found, then we have no saved configs
	if len(keyList) == 0 {
		success(w, "No older versions of the Alertmanager configuration were found.")
		return
	}
	for k := range keyList {
		resultMap["versions"] = append(resultMap["versions"], strings.Replace(keyList[k], AlertmanagerConfigFileName+"-", "", 1))
	}
	result, _ = json.MarshalIndent(resultMap, "", "\t")
	successBytes(w, result)

}

func (k *K8s) PutAlertmanagerConfig(w http.ResponseWriter, r *http.Request) {
	b, e := ioutil.ReadAll(r.Body)
	if e != nil {
		internalError(w, "Unable to read request Body: "+e.Error())
		return
	}

	// Validate the user-provided body
	amOut, e := checkAlertmanagerConfig(b)
	log(LevelInfo, "%s\n", amOut)
	if e != nil {
		badRequest(w, "Alertmanager configuration was not updated.  Failed to validate config with amtool: "+string(amOut)+" :ErrorMsg: "+e.Error())
		return
	}

	// Get the configmaps
	currentConfigMapName, currentConfigMap, err := k.getConfigMapByPath(AlertmanagerConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertmanager-config ConfigMap: %v", err))
		return
	}
	savedConfigMapName, savedConfigMap, err := k.getConfigMapByPath(AlertmanagerVersionsConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read alertmanager-config-versions ConfigMap: %v", err))
		return
	}

	// Special check... did the user make any changes?  If not, take no action and exit
	if currentConfigMap[AlertmanagerConfigFileName] == string(b) {
		success(w, "The provided body is identical to the current AlertManager configuration. No action will be taken.")
		return
	}

	// Copy the current alertmanager.yml to the versons ConfigMap
	// Default name for all keys is:  "alertmanager.yml-TIMESTAMP"
	timeNow := time.Now().UTC()
	keyName := AlertmanagerConfigFileName + "-" + timeNow.Format(Layout)

	// One-time step:  need to initialize the empty map the first time
	if savedConfigMap == nil {
		savedConfigMap = make(map[string]string)
	}
	savedConfigMap[keyName] = currentConfigMap[AlertmanagerConfigFileName]

        // How many backups do we have?  Do we need to delete any old ones?
        // K8S configmaps have limited space; very large configs can fill the versions configMap.
        keyList := k.sortKeysFromConfigMap(savedConfigMap, AlertmanagerConfigFileName)

        // If we have more than MaxBackupFiles, check to see if any need to be removed...
        if len(keyList) > MaxBackupFiles {
                for j := range keyList {

                        // We want to keep at least MaxBackupFiles...
                        if j < MaxBackupFiles {
                                continue
                        }

                        // Delete any backups that are MaxBackupHours or older.
                        if k.isOldVersion(keyList[j], AlertmanagerConfigFileName, timeNow) {
                        	delete(savedConfigMap, keyList[j])
                        }
		}
	}

	// Okay, updating the versions configMap is completed.  Save it!
	e = k.updateConfigMapByName(savedConfigMap, savedConfigMapName)
	if e != nil {
		internalError(w, "Unable to save a backup of alertmanager.yml to alertmanager-config-versions ConfigMap. "+e.Error())
		return
	}

	// Finally, update the current Configmap with the new version (validated) provided by the user
	currentConfigMap[AlertmanagerConfigFileName] = string(b)
	e = k.updateConfigMapByName(currentConfigMap, currentConfigMapName)
	if e != nil {
		internalError(w, "Unable to update alertmanager-config ConfigMap: "+e.Error())
		return
	}
	// returning HTTP status "202: Accepted".
	// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
	// before the response is sent.
	accepted(w, "The Alertmanager configuration is being updated.")
	return
}

func checkAlertmanagerConfig(b []byte) ([]byte, error) {

	tf, e := saveDataToTempFile(b)
	if e != nil {
		log(LevelError, "failed to create temp file: %v \n", e)
		return nil, e
	}

	defer os.Remove(tf.Name())

	amtoolCommand := execute(amtoolPath, "check-config", tf.Name())
	amtoolOutput, err := amtoolCommand.CombinedOutput()
	if err != nil {
		log(LevelDebug, "%s check-config %s failed: (%s) %v\n",
			amtoolPath, tf.Name(), amtoolOutput, err)
	}
	return amtoolOutput, err
}
