// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"k8s.io/client-go/tools/remotecommand"

	"errors"
	"math/rand"
	"strconv"

	"github.com/Jeffail/gabs/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// LevelAll is all log level.
const LevelAll = 0

// LevelDebug is debug log level.
const LevelDebug = 1

// LevelInfo is info log level.
const LevelInfo = 2

// LevelError is error log level.
const LevelError = 3

// LevelFatal is fatal log level.
const LevelFatal = 4

// LevelMin is minimum log level.
const LevelMin = LevelAll

// LevelMax is maximum log level.
const LevelMax = LevelFatal

const defaultWaitTime = 10 * time.Second

var (
	levelNames               = [...]string{"FINEST", "DEBUG", "INFO", "ERROR", "FATAL"}
	errCanaryWithInvalidName = errors.New("Error: Canary with invalid Name ")
)

// EndpointBackoffSchedule is the schedule used for endpoint polling.
var EndpointBackoffSchedule = wait.Backoff{
	Steps:    10,
	Duration: 3 * time.Second,
	Factor:   2.0,
	Jitter:   0.0,
}

// A cheesy logger until we identify a decent Golang logging package.
func log(msgLevel int, format string, args ...interface{}) {
	if msgLevel >= debugLevel {
		leader := fmt.Sprintf("cirith [%s] %s: ",
			time.Now().UTC().Format(time.RFC3339), levelName(msgLevel))
		fmt.Fprintf(os.Stdout, leader+format+"\n", args...)
	}
}

func levelName(level int) string {
	if level < LevelMin {
		level = LevelMin
	}
	if level > LevelMax {
		level = LevelMax
	}
	return levelNames[level]
}

func execute(cmd string, args ...string) *exec.Cmd {
	concatenatedArgs := strings.Join(args, " ")
	log(LevelInfo, "execute: %s %s", cmd, concatenatedArgs)
	return exec.Command(cmd, args...)
}

func badRequest(w http.ResponseWriter, s string) {
	log(LevelInfo, "400 Bad Request: %s", s)
	w.WriteHeader(400)
	w.Write([]byte(s + "\r\n"))
}

//The server understood the request but refuses to authorize it.
func forbiddenError(w http.ResponseWriter, s string) {
	log(LevelInfo, "403 Forbidden Error: %s", s)
	w.WriteHeader(403)
	w.Write([]byte(s + "\r\n"))
}

//The requested resource could not be found.
func notFoundError(w http.ResponseWriter, s string) {
	log(LevelInfo, "404 Not Found: %s", s)
	w.WriteHeader(404)
	w.Write([]byte(s + "\r\n"))
}

//The request could not be completed due to a conflict with the current state of the target resource.
func conflictError(w http.ResponseWriter, s string) {
	log(LevelInfo, "409 Conflict Error: %s", s)
	w.WriteHeader(409)
	w.Write([]byte(s + "\r\n"))
}

func internalError(w http.ResponseWriter, s string) {
	log(LevelError, "500 Internal Server Error: %s", s)
	w.WriteHeader(500)
	w.Write([]byte(s + "\r\n"))
}

func serviceUnavailable(w http.ResponseWriter, s string) {
	log(LevelError, "503 Service Temporarily Unavailable: %s", s)
	w.WriteHeader(503)
	w.Write([]byte(s + "\r\n"))
}

func notImplemented(w http.ResponseWriter) {
	log(LevelInfo, "501 Not Implemented")
	w.WriteHeader(501)
	w.Write([]byte("Not Implemented\r\n"))
}

func success(w http.ResponseWriter, s string) {
	log(LevelInfo, "200 OK: %s", s)
	w.WriteHeader(200) // yes, it's the default
	w.Write([]byte(s + "\r\n"))
}

func accepted(w http.ResponseWriter, s string) {
	log(LevelInfo, "202 Accepted: %s", s)
	w.WriteHeader(202)
	w.Write([]byte(s + "\r\n"))
}

// Only use this to avoid a unnecessary conversion
func successBytes(w http.ResponseWriter, bytes []byte) { // it bites ;-)
	log(LevelInfo, "200 OK: (long content not logged)")
	w.WriteHeader(200) // yes, it's the default
	w.Write(bytes)
}

func saveDataToTempFile(data []byte) (*os.File, error) {
	rand.Seed(time.Now().UnixNano())

	tf, e := os.Create(os.TempDir() + "/cirith-" + strconv.Itoa(rand.Intn(32)))
	if e != nil {
		log(LevelError, "cannot create temp file: %v\n", e.Error())
		return nil, e
	}

	defer tf.Close()

	n2, e := tf.Write(data)
	if e != nil {
		log(LevelError, "cannot write temp file: %v\n", e.Error())
		return nil, e
	}

	log(LevelDebug, "wrote %d bytes into %s\n", n2, tf.Name())

	return tf, e
}

// Retry executes the provided function repeatedly, retrying until the function
// returns done = true, or exceeds the given timeout.
func Retry(schedule wait.Backoff, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(schedule, func() (bool, error) {
		done, err := fn()
		if err != nil {
			lastErr = err
		}
		return done, nil // we never let an error thrown by the function cause the retry loop to end
	})
	if lastErr != nil {
		err = lastErr
	}
	return err
}

func (k *K8s) getNameFromVMISpec() string {
	var value = ""

	vmi, e := k.getVMIJson()
	if e != nil {
		log(LevelError, "Unable to get Verrazzano Monitoring Instance (VMI) Spec Json.")
	} else {
		if vmi.Exists("metadata", "name") {
			value = vmi.Search("metadata", "name").Data().(string)
		} else {
			log(LevelError, "Unable to get name from Verrazzano Monitoring Instance (VMI) Spec.")
		}

	}
	return value
}

// Set the storage Limits from the current VMI spec
func (k *K8s) getStorageLimitFromVMISpec(component, limit string) string {
	var value = ""

	vmi, e := k.getVMIJson()
	if e != nil {
		log(LevelError, "Unable to get Verrazzano Monitoring Instance (VMI) Spec Json.")
	} else {
		if vmi.Exists("spec", component, "resources", limit) {
			value = vmi.Search("spec", component, "resources", limit).Data().(string)
		} else {
			log(LevelError, "Unable to get disk size limits from Verrazzano Monitoring Instance (VMI) Spec.")
		}

	}
	return value
}

// Get the storage capacity from the current VMI spec
func (k *K8s) getStorageCapacityFromVMISpec(component string) string {
	var value = ""

	vmi, e := k.getVMIJson()
	if e != nil {
		log(LevelError, "Unable to get Verrazzano Monitoring Instance (VMI) Spec Json.")
	} else {
		if vmi.Exists("spec", component, "storage", "size") {
			value = vmi.Search("spec", component, "storage", "size").Data().(string)
		} else {
			log(LevelError, "Unable to get disk capacity from Verrazzano Monitoring Instance (VMI) Spec.")
		}

	}
	return value
}

// Calculates the current storage capacity and returns either double/halve
//  takes a string for current storage capacity and and operation {double, halve}
//  double returns an error if current capacity is already at maxSize,
//      double is limited to a maximum value of maxSize
//  halve returns and error if current capacity is already at minSizeGb
//       halve is limited to a minimum of minSizeGb
func (k *K8s) modifyStorageCapacity(component string, factor float32) (string, error) {

	var maxSize resource.Quantity
	var minSize resource.Quantity

	currentStorageCapacity := k.getStorageCapacityFromVMISpec(component)

	currentQuantity, err := resource.ParseQuantity(currentStorageCapacity)
	if err != nil {
		log(LevelError, "Error: %v", err)
		return currentStorageCapacity, errors.New("Failed to get current storage capacity: " + err.Error())
	}
	log(LevelDebug, "currentSize:%v", currentQuantity)

	maxSize, err = resource.ParseQuantity(k.getStorageLimitFromVMISpec(component, maxSizeDisk))
	if err != nil {
		log(LevelError, "Using default max size: %v ,%v", defaultMaxSize, err)
		tmpQuantity := resource.NewQuantity(defaultMaxSize*1024*1024*1024, resource.BinarySI)
		tmpQuantity.DeepCopyInto(&maxSize)
	}
	log(LevelDebug, "maxSize:%v", maxSize)

	minSize, err = resource.ParseQuantity(k.getStorageLimitFromVMISpec(component, minSizeDisk))
	if err != nil {
		log(LevelError, "Using default min size: %v, %v", defaultMinSize, err)
		tmpQuantity := resource.NewQuantity(defaultMinSize*1024*1024*1024, resource.BinarySI)
		tmpQuantity.DeepCopyInto(&minSize)
	}
	log(LevelDebug, "minSize:%v", minSize)

	newSize := resource.NewQuantity(int64(float32(currentQuantity.Value())*factor), resource.BinarySI)
	log(LevelDebug, "newSize:%v", newSize)

	if newSize.Cmp(maxSize) == 1 {
		log(LevelError, "Request exceeds maximum disk size: %s", maxSize.String())
		return currentStorageCapacity, errors.New("Request exceeds the maximum disk size " + maxSize.String())
	}
	if newSize.Cmp(minSize) == -1 {
		log(LevelError, "Request exceeds minimum disk size allowed: %s", minSize.String())
		return currentStorageCapacity, errors.New("Request exceeds the minimum disk size " + minSize.String())
	}

	return newSize.String(), nil
}

// getVMIJson retrieves the current Verrazzano Monitoring Instance (VMI) from k8s as a JSON entity
func (k *K8s) getVMIJson() (*gabs.Container, error) {
	result, err := k.RestClient.Get().Resource(VMIPlural).Namespace(namespace).Name(vmiName).Do(context.TODO()).Raw()
	if err != nil {
		return nil, err
	}
	vmiJSON, err := gabs.ParseJSON(result)
	if err != nil {
		return nil, err
	}
	return vmiJSON, nil
}

// Updates the given Verrazzano Monitoring Instance (VMI) (specified as a JSON entity) in k8s
func (k *K8s) updateVMIJson(vmi *gabs.Container) error {
	_, err := k.RestClient.Put().Resource(VMIPlural).Namespace(namespace).Name(vmiName).Body(vmi.Bytes()).Do(context.TODO()).Raw()
	if err != nil {
		return err
	}
	return nil
}

func (k *K8s) getConfigMapByName(name string) (map[string]string, error) {
	cm, err := k.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cm.Data, nil
}

func (k *K8s) getSecretByName(name string) (map[string][]byte, error) {
	secret, err := k.ClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// getConfigMapByPath looks up the ConfigMap at the given path in the spec, and returns the name and the ConfigMap.
// Note that the ConfigMap name is required if you want to update the ConfigMap data.
func (k *K8s) getConfigMapByPath(path string) (string, map[string]string, error) {
	vmi, err := k.getVMIJson()
	if err != nil {
		log(LevelError, "Unable to get Verrazzano Monitoring Instance (VMI) JSON: %v", err)
		return "", nil, err
	}
	configMapName := vmi.Path(path).Data().(string)
	cm, err := k.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		log(LevelError, "Unable to get ConfigMap %s: %v", configMapName, err)
		return "", nil, err
	}
	return configMapName, cm.Data, nil
}

func (k *K8s) updateConfigMapByName(updatedMap map[string]string, name string) error {
	cm, err := k.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		return err
	}
	cm.Data = updatedMap
	if _, err = k.ClientSet.CoreV1().ConfigMaps(namespace).Update(context.TODO(), cm, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// Poll until the configmap update has been applied or we reach the timeout.
	err = wait.PollImmediate(300*time.Millisecond, defaultWaitTime, func() (done bool, err error) {
		cm, err = k.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		// stop polling if we get an error
		if err != nil {
			return true, err
		}
		// If we just emptied the map, data will be nil (not empty).
		if len(updatedMap) == 0 && cm.Data == nil {
			return true, nil
		}
		// Otherwise compare.
		if reflect.DeepEqual(cm.Data, updatedMap) {
			return true, nil
		}
		fmt.Println("configmap ", cm.Name, " in namespace ", cm.Namespace, " does not appear updated: ", cm.Data)
		return false, nil
	})

	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("verification of the updated configuration timed out %v", defaultWaitTime)
		}
		return err
	}

	return nil
}

// Given a configmap, extract all of the keys that match a given filename.
// Return a list of those keys sorted by timestamp.
func (k *K8s) sortKeysFromConfigMap(configMap map[string]string, fileName string) []string {

	keyList := make([]string, 0, len(configMap))
	for k := range configMap {
		// If a fileName was provided, ignore any keys that don't match the provided filename
		if fileName != "" && strings.HasPrefix(k, fileName) {
			keyList = append(keyList, k)
		}
	}

	// Sort the list of keys by timestamp
	sort.Slice(keyList, func(i, j int) bool {
		time1, _ := time.Parse(Layout, strings.Replace(keyList[i], fileName+"-", "", 1))
		time2, _ := time.Parse(Layout, strings.Replace(keyList[j], fileName+"-", "", 1))
		return time1.After(time2)
	})
	return keyList
}

// Check if a given key is MaxBackupHours old or older...
func (k *K8s) isOldVersion(key string, fileName string, timeNow time.Time) bool {

	keyTime, _ := time.Parse(Layout, strings.Replace(key, fileName+"-", "", 1))
	diff := timeNow.Sub(keyTime)
	hours := int(diff.Hours())
	if hours >= MaxBackupHours {
		return true
	}
	return false
}

func (k *K8s) updateSecretByName(updatedSecret map[string][]byte, name string) error {
	secret, err := k.ClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		return err
	}

	secret.Data = updatedSecret
	if _, err = k.ClientSet.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{}); err != nil {
		return err
	}

	// Poll until the secret update has been applied or we reach the timeout
	err = wait.PollImmediate(300*time.Millisecond, defaultWaitTime, func() (done bool, err error) {
		secret, err := k.ClientSet.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		// stop polling if we get an error
		if err != nil {
			return true, err
		}
		// If we just emptied the map, data will be nil (not empty).
		if len(updatedSecret) == 0 && secret.Data == nil {
			return true, nil
		}
		if reflect.DeepEqual(secret.Data, updatedSecret) {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("verification of the updated secret timed out %v", defaultWaitTime)
		}
		return err
	}

	return nil
}

//Rules names may contain only ASCII A-Za-z0-9-_
func validateName(proposedName string) error {
	s := strings.Replace(proposedName, "-", "", -1)
	match, err := regexp.MatchString("[[:word:]]", s)
	if err == nil && match {
		return nil
	}
	return errCanaryWithInvalidName
}

// Usage returns the usage for the API.
func Usage(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "API server for \"VMI\" ! <a href=\"https://github.com/verrazzano/verrazzano-monitoring-instance-api/blob/master/README.md\">README</a>")
}

// See <https://kubernetes.io/docs/concepts/overview/working-with-objects/names/> for details
var configMapKeyNameRegex = regexp.MustCompile("^[-._a-zA-Z0-9]+$")

// ValidateConfigMapKeyName validates a given configmap key.
func ValidateConfigMapKeyName(keyName string) error {
	if len(keyName) == 0 || len(keyName) > 253 {
		return errors.New("ConfigMap key names must be between 1 and 253 characters long")
	}
	if len(configMapKeyNameRegex.FindString(keyName)) == 0 {
		return errors.New("invalid ConfigMap key name")
	}
	return nil
}

// Close calls close on the given Closer, and catches and logs any errors.  Useful to call deferred
func Close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log(LevelError, "error closing: %v", err)
	}
}

// makeTar writes a Tar, with one entry for the given data and name, to the given writer
func makeTar(data []byte, entryName string, writer io.Writer) error {
	tarWriter := tar.NewWriter(writer)
	log(LevelDebug, "Created new tar writer")
	defer Close(tarWriter)
	hdr := tar.Header{
		Name: entryName,
		Mode: 0600,
		Size: int64(len(data)),
	}
	log(LevelDebug, "Created new tar header")
	if err := tarWriter.WriteHeader(&hdr); err != nil {
		log(LevelError, "problem writing tar header: %v", err)
		return err
	}
	log(LevelDebug, "Wrote tar header")
	if _, err := tarWriter.Write(data); err != nil {
		log(LevelError, "problem writing tar body: %v", err)
		return err
	}
	log(LevelInfo, "Created tar for %s (%d bytes)", entryName, len(data))
	return nil
}

// within returns true if t1 is within +/- threshold of t2
func within(t1, t2 time.Time, threshold time.Duration) bool {
	// t1 should not be later than t2
	if t1.After(t2) {
		t1, t2 = t2, t1
	}
	diff := t2.Sub(t1)
	if diff <= threshold {
		return true
	}
	return false
}

// copyFile copies the given data to the given directory in the given pod and container.  Note that the destDir
// must already exist on the container
func (k *K8s) copyFile(data []byte, fileName string, destDir string, podName string, containerName string) error {
	log(LevelDebug, "Copying file %s to %s:%s", fileName, podName, containerName)

	// Following approach used by implementation of `kubectl cp`
	// <https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/cp/cp.go>, create a pipe, and then a
	// goroutine that writes to the pipe's writer a tar containing the rule data.  This will feed the reader provided
	// below to the POST request sent to the pod.
	reader, writer := io.Pipe()

	go func() {
		defer Close(writer)
		err := makeTar(data, fileName, writer)
		if err != nil {
			log(LevelError, "Problem making tar: %v", err)
		}
	}()

	request := k.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName).
		Param("command", "tar").
		Param("command", "xf").
		Param("command", "-").
		Param("command", "-C").
		Param("command", destDir).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true")
	executor, err := remotecommand.NewSPDYExecutor(k.Config, "POST", request.URL())
	if err != nil {
		log(LevelError, "problem executing POST request to copy file: %s", err.Error())
		return err
	}
	var (
		execOut1 bytes.Buffer
		execErr1 bytes.Buffer
	)
	err = executor.Stream(remotecommand.StreamOptions{Stdin: reader, Stdout: &execOut1, Stderr: &execErr1, Tty: false})
	if err != nil {
		log(LevelError, "Copying file %s to %s:%s failed: %v", fileName, podName, containerName, err)
		return err
	}
	log(LevelInfo, "Copied file %s to %s:%s", fileName, podName, containerName)
	return nil
}

// deleteFile deletes the file at the given path on the given pod/container.
func (k *K8s) deleteFile(filePath string, podName string, containerName string) error {
	log(LevelDebug, "Deleting file %s on %s:%s", filePath, podName, containerName)

	request := k.ClientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName).
		Param("command", "rm").
		Param("command", filePath).
		Param("stdin", "false").
		Param("stdout", "true").
		Param("stderr", "true")
	executor, err := remotecommand.NewSPDYExecutor(k.Config, "POST", request.URL())
	if err != nil {
		log(LevelError, "problem executing POST request to send delete-file command: %s", err.Error())
		return err
	}
	var (
		execOut1 bytes.Buffer
		execErr1 bytes.Buffer
	)
	err = executor.Stream(remotecommand.StreamOptions{Stdout: &execOut1, Stderr: &execErr1, Tty: false})
	if err != nil {
		log(LevelError, "Deleting file %s:%s:%s failed: %v", podName, containerName, filePath, err)
		return err
	}
	log(LevelInfo, "Deleted file %s on %s:%s", filePath, podName, containerName)
	return nil
}

// getNodeIPs returns a list of public IPs for all nodes in the configured cluster.
func (k *K8s) getNodeIPs() ([]string, error) {

	// Construct a map of node names => public IPs for easy lookup below
	nodes, err := k.ClientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log(LevelError, "Unable to list nodes: %v", err)
		return []string{}, err

	}
	ipList := []string{}
	if nodes != nil {
		for _, node := range nodes.Items {
			nodeAddresses := node.Status.Addresses
			for _, nodeAddress := range nodeAddresses {
				if nodeAddress.Type == corev1.NodeExternalIP {
					ipList = append(ipList, nodeAddress.Address)
				}
			}
		}
	}
	return ipList, nil

}

func sendRequest(action, myURL, host string, headers map[string]string, payload string, reqUserName string, reqPassword string) (*http.Response, string, error) {
	var err error

	tr := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 300 * time.Second}
	tURL := url.URL{}

	// Set proxy for http client - needs to be done unless using localhost
	proxyURL := os.Getenv("http_proxy")
	if proxyURL != "" {
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}

	// fmt.Printf(" --> Request: %s - %s\n", action, myURL)
	req, err := http.NewRequest(action, myURL, strings.NewReader(payload))
	if err != nil {
		return nil, "", err
	}

	// Add any headers to the request
	for k := range headers {
		req.Header.Add(k, headers[k])
	}

	req.Host = host

	req.SetBasicAuth(reqUserName, reqPassword)

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// Extract the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(body), err
}

// WaitForEndpointAvailable waits for the given endpoint to become available.
func WaitForEndpointAvailable(action, myURL, host string, headers map[string]string, payload string, reqUserName string, reqPassword string) error {
	var err error
	expectedStatusCode := http.StatusOK
	log(LevelInfo, "Waiting for %s to reach status code %d...\n", myURL, expectedStatusCode)
	startTime := time.Now()

	err = Retry(EndpointBackoffSchedule, func() (bool, error) {
		resp, _, reqErr := sendRequest(action, myURL, host, headers, payload, reqUserName, reqPassword)
		if reqErr != nil {
			return false, reqErr
		}
		if resp.StatusCode == expectedStatusCode {
			return true, nil
		}
		return false, nil
	})
	log(LevelInfo, "Wait time: %s \n", time.Since(startTime))
	if err != nil {
		return err
	}
	return nil
}

// RestartPodsByLabel simulates a restart by (1) deleting all pods (sequentially) that have the given label, then (2)
// waiting up to waitTime for at least one *new* pod to be created and reach the READY status
// (PodPhase == PodRunning && [*]ContainerStatus.Ready == true). Note that even through we are ensure that the containers
// and pod are running, we cannot ensure the pod is actually ready to service requests.
// See docs for more information: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/
func (k *K8s) RestartPodsByLabel(label string, waitTime time.Duration, skipValidation string) error {

	// Following is a hack to skip validation during unit testing, when no container is running
	vmi, err := k.getVMIJson()
	if err != nil {
		log(LevelError, "Unable to get Verrazzano Monitoring Instance (VMI) JSON: %v", err)
		return err
	}

	skipValidationObj := vmi.Path(skipValidation).Data()
	if skipValidationObj == nil || !skipValidationObj.(bool) {
		pods, err := k.getPodsByLabel(label)
		if err != nil {
			log(LevelError, "RestartPodsByLabel() %s: list pods failed %v", err.Error())
			return err
		}
		if len(pods.Items) == 0 {
			// This is an error. In case of Restart pod, a pod should exist.
			errMessage := fmt.Sprintf("no pods with label %s found to restart", label)
			log(LevelError, errMessage)
			return errors.New(errMessage)
		}
		err = k.deletePods(pods)
		if err != nil {
			log(LevelError, "RestartPodsByLabel() %s: delete pods failed %v", err.Error())
			return err
		}
	}
	return nil
}

// getPodsByLabel returns a PodList that match the specified label.
func (k *K8s) getPodsByLabel(label string) (*corev1.PodList, error) {
	pods, err := k.ClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

// getReadyPodsByLabel returns a PodList that match the specified label *and* whose overall PodPhase is equal to
// PodRunning. When allContainersReady is true, it will also ensure that each container in the pod is Ready in addition
// to verifying the PodPhase.
func (k *K8s) getReadyPodsByLabel(label string, allContainersReady bool) (*corev1.PodList, error) {
	log(LevelDebug, "Looking for ready pods with label %s", label)
	var podsInPhase []corev1.Pod
	podList, err := k.ClientSet.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}
	for i := range podList.Items {
		if corev1.PodRunning == podList.Items[i].Status.Phase {
			log(LevelDebug, "pod %s is running", podList.Items[i].Name)
			containersReady := true
			if allContainersReady {
				for j := range podList.Items[i].Spec.Containers {
					if !podList.Items[i].Status.ContainerStatuses[j].Ready {
						log(LevelDebug, "not accepting pod %s since container %d is NOT ready", podList.Items[i].Name, j)
						containersReady = false
					}
				}
				// allContainersReady is true, only add if containers were all found to be ready
				if containersReady {
					log(LevelDebug, "accepting pod %s since all its containers are ready", podList.Items[i].Name)
					podsInPhase = append(podsInPhase, podList.Items[i])
				}
			} else {
				// allContainersReady was false, we only have to evaluate pod is in phase
				log(LevelDebug, "accepting pod %s since it is PodRunning", podList.Items[i].Name)
				podsInPhase = append(podsInPhase, podList.Items[i])
			}

		} else {
			log(LevelDebug, "not accepting pod %s since it is NOT PodRunning", podList.Items[i].Name)
		}
	}
	podList.Items = podsInPhase
	return podList, nil
}

// deletePods makes a request to delete all pods in the podList sequentially without waiting for each
// request to be started (or completed). Even though the each pod is deleted sequentially, pods that rely on at least
// one of the replicas being up at all times for data availability (e.g. AlertManager) should use the deletePod function.
func (k *K8s) deletePods(podList *corev1.PodList) error {
	for _, pod := range podList.Items {
		log(LevelInfo, "deleting pods %s in namespace %s\n", pod.Name, namespace)
		if err := k.ClientSet.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err == nil {
			return err
		}
	}
	return nil
}

// deletePod deletes a single pod with a given name
func (k *K8s) deletePod(podName string) error {
	log(LevelInfo, "deletePod %s in namespace %s\n", podName, namespace)
	return k.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
}
