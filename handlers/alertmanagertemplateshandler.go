// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"sort"
	"strings"
)

// GetAllAlertmanagerTemplatesFileNames returns all Alert Manager template files.
func (k *K8s) GetAllAlertmanagerTemplatesFileNames(w http.ResponseWriter, r *http.Request) {

	_, configMap, err := k.getConfigMapByPath(AlertmanagerTemplatesConfigMapPath)
	if err != nil {
		internalError(w, fmt.Sprintf("Unable to read Alertmanager template ConfigMap: %v", err))
		return
	}

	templateNames := make([]string, len(configMap))
	i := 0
	for k := range configMap {
		templateNames[i] = k
		i++
	}
	sort.Strings(templateNames)
	success(w, strings.Join(templateNames, "\n"))
}

// GetAlertmanagerTemplate returns a requested Alert Manager template file.
func (k *K8s) GetAlertmanagerTemplate(w http.ResponseWriter, r *http.Request) {
	fileName := path.Base(r.URL.Path)
	err := validateName(fileName)
	if err != nil {
		badRequest(w, "ERROR: Invalid File Name: "+err.Error())
		return
	}

	vmi, e := k.getVMIJson()
	amTemplateMapName := vmi.Path(AlertmanagerTemplatesConfigMapPath).Data().(string)
	templatesMap, e := k.getConfigMapByName(amTemplateMapName)
	if e != nil {
		internalError(w, "Unable to read ConfigMap: "+amTemplateMapName+", "+e.Error())
		return
	}
	if e != nil {
		internalError(w, "Unable to read ConfigMap: "+amTemplateMapName+", "+e.Error())
		return
	}

	for k, v := range templatesMap {
		if k == fileName {
			log(LevelDebug, "%s", "Found existing file in Map: "+amTemplateMapName+", "+fileName)
			success(w, v)
			return
		}
	}
	badRequest(w, "Did not find any template with name: "+amTemplateMapName+", "+fileName)
}

// DeleteAlertmanagerTemplate deletes a requested Alert Manager template file.
func (k *K8s) DeleteAlertmanagerTemplate(w http.ResponseWriter, r *http.Request) {
	fileName := path.Base(r.URL.Path)
	err := validateName(fileName)
	if err != nil {
		badRequest(w, "ERROR: Invalid File Name: "+err.Error())
		return
	}

	vmi, e := k.getVMIJson()
	amTemplateMapName := vmi.Path(AlertmanagerTemplatesConfigMapPath).Data().(string)
	templatesMap, e := k.getConfigMapByName(amTemplateMapName)
	if e != nil {
		internalError(w, "Unable to read ConfigMap: "+e.Error())
		return
	}

	for j := range templatesMap {
		if j == fileName {
			log(LevelDebug, "%s", "Found existing file in Map: "+fileName)
			delete(templatesMap, j)
			e = k.updateConfigMapByName(templatesMap, amTemplateMapName)
			if e != nil {
				internalError(w, "Unable to update ConfigMap: "+e.Error())
				return
			}
			// returning HTTP status "202: Accepted".
			// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
			// before the response is sent.  I.e. a client might send a DELETE request to delete a template, receive a 200 response,
			// and quickly send a GET request for the list of all templates, and receive a response that still includes the template.
			accepted(w, "Deleting template file: "+amTemplateMapName+", "+fileName)
			return
		}
	}
	badRequest(w, "Did not find any template to delete with name: "+fileName)
}

// PutAlertmanagerTemplate adds a requested Alert Manager template file.
func (k *K8s) PutAlertmanagerTemplate(w http.ResponseWriter, r *http.Request) {
	b, e := ioutil.ReadAll(r.Body)
	if e != nil {
		internalError(w, "ERROR: Unable to read request Body: "+e.Error())
		return
	}
	fileName := path.Base(r.URL.Path)
	if fileName == "" {
		badRequest(w, "ERROR: Did not pass mandatory parameter filename, Please pass /v1/alertmanager/addtemplate?filename=<name.templates>")
		return
	}

	e = validateName(fileName)
	if e != nil {
		badRequest(w, "ERROR: Invalid File Name: "+e.Error())
		return
	}

	if !strings.HasSuffix(fileName, ".tmpl") {
		badRequest(w, "ERROR: Filename should end with .tmpl only")
		return
	}

	vmi, e := k.getVMIJson()
	if e != nil {
		internalError(w, "Unable to GET Verrazzano Monitoring Instance: "+e.Error())
		return
	}

	amTemplateMapName := vmi.Path(AlertmanagerTemplatesConfigMapPath).Data().(string)
	templatesMap, e := k.getConfigMapByName(amTemplateMapName)
	if e != nil {
		internalError(w, "Unable to read ConfigMap: "+amTemplateMapName+", "+e.Error())
		return
	}
	for j := range templatesMap {
		if j == fileName {
			log(LevelDebug, "%s", "Found existing file in Map: "+fileName)
			templatesMap[j] = string(b)
			e = k.updateConfigMapByName(templatesMap, amTemplateMapName)
			if e != nil {
				internalError(w, "Unable to update ConfigMap: "+amTemplateMapName+", "+e.Error())
				return
			}
			// returning HTTP status "202: Accepted".
			// Changes to ConfigMap instances are eventually propagated to the consuming containers, but this might not complete
			// before the response is sent.  I.e. a client might send a PUT request to create or update a template, receive a 200
			// response, and quickly send a GET request for that template but receive a 404 in the case of a new template, or 200 with
			// the previous version in the case of an existing template.
			accepted(w, "Updating existing template in Map: "+amTemplateMapName+", "+fileName)
			return
		}
	}

	//We did not find an existing template, So add it has a new Rule.
	newKey := fileName
	newValue := string(b)

	if templatesMap == nil {
		templatesMap = make(map[string]string)
		templatesMap[newKey] = newValue
	} else {
		templatesMap[newKey] = newValue
	}

	e = k.updateConfigMapByName(templatesMap, amTemplateMapName)
	if e != nil {
		internalError(w, "Unable to update ConfigMap: "+amTemplateMapName+", "+e.Error())
		return
	}
	// returning HTTP status "202: Accepted".
	accepted(w, "Adding new template file name: "+amTemplateMapName+", "+fileName)

}
