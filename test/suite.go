package test

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-test/pkg/onostest"
	"sync"
)

// Suite is the onos-config test suite
type Suite struct {
	test.Suite
}

// InstallUmbrella creates a helm install command for an onos-umbrella instance
func (s *Suite) InstallUmbrella() *helm.InstallCmd {
	return s.Helm().
		Install("onos-umbrella", "onos-umbrella").
		RepoURL(onostest.OnosChartRepo).
		Set("onos-topo.image.tag", "latest").
		Set("onos-config.image.tag", "latest").
		Set("onos-config-model.image.tag", "latest")
}

func iterAsync(n int, f func(i int) error) error {
	wg := sync.WaitGroup{}
	asyncErrors := make(chan error, n)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(j int) {
			err := f(j)
			if err != nil {
				asyncErrors <- err
			}
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(asyncErrors)
	}()

	for err := range asyncErrors {
		return err
	}
	return nil
}
