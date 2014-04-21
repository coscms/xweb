package httpsession

import "errors"

type AfterCreatedListener interface {
	OnAfterCreated(*Session)
}

type BeforeReleaseListener interface {
	OnBeforeRelease(*Session)
}

func (manager *Manager) AddListener(listener interface{}) error {
	switch listener.(type) {
	case AfterCreatedListener:
		manager.afterCreatedListeners[listener.(AfterCreatedListener)] = true
	case BeforeReleaseListener:
		manager.beforeReleaseListeners[listener.(BeforeReleaseListener)] = true
	default:
		return errors.New("Unknow listener type")
	}
	return nil
}

func (manager *Manager) RemoveListener(listener interface{}) error {
	switch listener.(type) {
	case AfterCreatedListener:
		delete(manager.afterCreatedListeners, listener.(AfterCreatedListener))
	case BeforeReleaseListener:
		delete(manager.beforeReleaseListeners, listener.(BeforeReleaseListener))
	default:
		return errors.New("Unknow listener type")
	}
	return nil
}
