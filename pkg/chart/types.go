package chart

import "k8s.io/helm/pkg/chartutil"

// Constants
const (
	APIV1                    string = "v1"
	APIV2                    string = "v2"
	ChartFilename            string = chartutil.ChartfileName
	ChartLockFilename        string = "Chart.lock"
	ValuesFilename           string = chartutil.ValuesfileName
	ValuesProductionFilename string = "values-production.yaml"
	RequirementsFilename     string = "requirements.yaml"
	RequirementsLockFilename string = "requirements.lock"
	ReadmeFilename           string = "README.md"
)


