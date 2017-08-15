package proxy

import (
	"fmt"
	"io/ioutil"
)

func LoadDataOnce(r *rudderProxy, istioContainerPath, istioInitPath string) error {
	r.dataSync.Lock()
	defer r.dataSync.Unlock()
	containerData, err := ioutil.ReadFile(istioContainerPath)
	if err != nil {
		return err
	}
	initData, err := ioutil.ReadFile(istioInitPath)
	if err != nil {
		return err
	}
	if len(containerData) == 0 {
		return fmt.Errorf("Container definition can't be empty, source file: %s", istioContainerPath)
	}
	r.istioContainerData = string(containerData)
	r.istioInitData = string(initData)
	return nil
}
