package utils

import "testing"

func TestReadCsv(t *testing.T) {
	ReadCsv(`model_baseline.csv`)
}

func TestMakeRandomJobQueue(t *testing.T) {
	MakeRandomJobQueue(`/root/cpn_controller/pkg/yaml_template`, `/root/cpn_controller/pkg/controller/yaml_queue`)
}
