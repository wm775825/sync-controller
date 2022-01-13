package controller

type Controller interface {
	Run(stopCh chan struct{}, sync bool) error
}
