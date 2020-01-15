package oam

import (
	"os"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Option func() error

type ControllerContext struct {
	mgr      ctrl.Manager
	l        *sync.RWMutex
	handlers map[SType][]Handler
	owns     map[SType][]runtime.Object
}

var (
	oamLog            = ctrl.Log.WithName("oam")
	controllerContext = ControllerContext{
		handlers: make(map[SType][]Handler),
		owns:     make(map[SType][]runtime.Object),
		l:        new(sync.RWMutex),
	}
)

func InitMgr(conf *rest.Config, options ctrl.Options) {
	if options.MetricsBindAddress == "" {
		// disable metrics
		options.MetricsBindAddress = "0"
	}
	m, err := ctrl.NewManager(conf, options)
	if err != nil {
		oamLog.Error(err, "unable to init manager")
		os.Exit(1)
	}
	controllerContext.mgr = m
}

func GetMgr() ctrl.Manager {
	return controllerContext.mgr
}

func RegisterHandlers(name SType, handlers ...Handler) {
	controllerContext.l.Lock()
	defer controllerContext.l.Unlock()
	controllerContext.handlers[name] = append(controllerContext.handlers[name], handlers...)
}

func Owns(name SType, owns ...runtime.Object) {
	controllerContext.l.Lock()
	defer controllerContext.l.Unlock()
	controllerContext.owns[name] = append(controllerContext.owns[name], owns...)
}
func getOwns(name SType) []runtime.Object {
	controllerContext.l.RLock()
	defer controllerContext.l.RUnlock()
	return controllerContext.owns[name]
}

func getHandlers(name SType) []Handler {
	controllerContext.l.RLock()
	defer controllerContext.l.RUnlock()
	return controllerContext.handlers[name]
}

func withSpec(tp SType) Option {
	return func() error {
		return (&Reconciler{
			specType:          tp,
			Client:            controllerContext.mgr.GetClient(),
			Log:               ctrl.Log.WithName("oma-controller").WithName(string(tp)),
			Scheme:            controllerContext.mgr.GetScheme(),
			ControllerContext: controllerContext,
		}).SetupWithManager(controllerContext.mgr)
	}

}

// WithComponent registers Component reconciler
func WithComponent() Option {
	return withSpec(STypeComponent)

}

// WithScpe registers Scpe reconciler
func WithScope() Option {
	return withSpec(STypeScope)
}

// WithWorkloadType registers WorkloadType reconciler
func WithWorkloadType() Option {
	return withSpec(STypeWorkloadType)
}

// WithTrait registers Trait reconciler
func WithTrait() Option {
	return withSpec(STypeTrait)
}

// WithApplicationConfiguration registers ApplicationConfiguration reconciler
func WithApplicationConfiguration() Option {
	return withSpec(STypeApplicationConfiguration)
}
func Run(options ...Option) error {
	for _, o := range options {
		if err := o(); err != nil {
			return err
		}
	}

	oamLog.Info("starting controller manager")
	if err := controllerContext.mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		oamLog.Error(err, "problem running controller manager")
		return err
	}
	return nil
}
