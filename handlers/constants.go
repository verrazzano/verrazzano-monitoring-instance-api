// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package handlers

import "net"

// These can be set from the command line via e.g. -promRulesFile <promRulesFilePath>

// ListenURL Cirith listener URL
var ListenURL string

var promtoolPath string
var staticPath string
var debugLevel = LevelInfo
var masterURL string
var kubeconfig string
var ociConfigFile string
var natGatewayIPs []net.IP
var vmiName string
var namespace string
var defaultMaxSize int64
var defaultMinSize int64
var backupBucket string

const minSizeDisk = "minSizeDisk"
const maxSizeDisk = "maxSizeDisk"

// Layout format of time stamp.
const Layout = "2006-01-02T15-04-05"

// MaxBackupFiles is max number of files for backup.
const MaxBackupFiles = 10

// MaxBackupHours is max hours for backup.
const MaxBackupHours = 48

// The follow are the only operator-dependent elements we rely on

// VMIGroup group name for an instance resource.
const VMIGroup = "verrazzano.io"

// VMIVersion version of instance resource.
const VMIVersion = "v1"

// VMIPlural plural name for an instance resource.
const VMIPlural = "verrazzanomonitoringinstances"

// VMIMetadataNamePath path for metadata name.
const VMIMetadataNamePath = "metadata.name"

// PrometheusConfigMapPath path for Prometheus configMap.
const PrometheusConfigMapPath = "spec.prometheus.configMap"

// PrometheusVersionsConfigMapPath path for Prometheus versions configMap.
const PrometheusVersionsConfigMapPath = "spec.prometheus.versionsConfigMap"

// PrometheusRulesConfigMapPath path for Prometheus rules configMap.
const PrometheusRulesConfigMapPath = "spec.prometheus.rulesConfigMap"

// PrometheusRulesVersionsConfigMapPath path for Prometheus rules versions configMap.
const PrometheusRulesVersionsConfigMapPath = "spec.prometheus.rulesVersionsConfigMap"

// AlertmanagerTemplatesConfigMapPath for Alert Manager configMap.
const AlertmanagerTemplatesConfigMapPath = "spec.alertmanager.templatesConfigMap"

// PrometheusConfigFileName file name of Prometheus config file.
const PrometheusConfigFileName = "prometheus.yml"

// K8sPublicIPAddressLabel label name for IP address.
const K8sPublicIPAddressLabel = "node.info/external.ipaddress"
