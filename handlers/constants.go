// Copyright (C) 2020, Oracle Corporation and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package handlers

import "net"

// These can be set from the command line via e.g. -promRulesFile <promRulesFilePath>
var ListenURL string
var amtoolPath string
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

const Layout = "2006-01-02T15-04-05"
const MaxBackupFiles = 10
const MaxBackupHours = 48

// The follow are the only operator-dependent elements we rely on
const VMIGroup = "verrazzano.io"
const VMIVersion = "v1"
const VMIPlural = "verrazzanomonitoringinstances"
const ResizePlural = "vmiresizes"
const VMIMetadataNamePath = "metadata.name"
const PrometheusConfigMapPath = "spec.prometheus.configMap"
const PrometheusVersionsConfigMapPath = "spec.prometheus.versionsConfigMap"
const PrometheusRulesConfigMapPath = "spec.prometheus.rulesConfigMap"
const PrometheusRulesVersionsConfigMapPath = "spec.prometheus.rulesVersionsConfigMap"
const AlertmanagerConfigMapPath = "spec.alertmanager.configMap"
const AlertmanagerVersionsConfigMapPath = "spec.alertmanager.versionsConfigMap"
const AlertmanagerTemplatesConfigMapPath = "spec.alertmanager.templatesConfigMap"

const PrometheusConfigFileName = "prometheus.yml"
const AlertmanagerConfigFileName = "alertmanager.yml"

const K8sPublicIpAddressLabel = "node.info/external.ipaddress"
