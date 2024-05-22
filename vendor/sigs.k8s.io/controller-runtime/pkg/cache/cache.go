/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/cache/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/internal/log"
)

var log = logf.RuntimeLog.WithName("object-cache")

// Cache knows how to load Kubernetes objects, fetch informers to request
// to receive events for Kubernetes objects (at a low-level),
// and add indices to fields on the objects stored in the cache.
type Cache interface {
	// Cache acts as a client to objects stored in the cache.
	client.Reader

	// Cache loads informers and adds field indices.
	Informers
}

// Informers knows how to create or fetch informers for different
// group-version-kinds, and add indices to those informers.  It's safe to call
// GetInformer from multiple threads.
type Informers interface {
	// GetInformer fetches or constructs an informer for the given object that corresponds to a single
	// API kind and resource.
	GetInformer(ctx context.Context, obj client.Object) (Informer, error)

	// GetInformerForKind is similar to GetInformer, except that it takes a group-version-kind, instead
	// of the underlying object.
	GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind, opts ...InformerGetOption) (Informer, error)

	// Start runs all the informers known to this cache until the context is closed.
	// It blocks.
	Start(ctx context.Context) error

	// WaitForCacheSync waits for all the caches to sync.  Returns false if it could not sync a cache.
	WaitForCacheSync(ctx context.Context) bool

	// Informers knows how to add indices to the caches (informers) that it manages.
	client.FieldIndexer
}

// Informer - informer allows you interact with the underlying informer.
type Informer interface {
	// AddEventHandler adds an event handler to the shared informer using the shared informer's resync
	// period.  Events to a single handler are delivered sequentially, but there is no coordination
	// between different handlers.
	AddEventHandler(handler toolscache.ResourceEventHandler)
	// AddEventHandlerWithResyncPeriod adds an event handler to the shared informer using the
	// specified resync period.  Events to a single handler are delivered sequentially, but there is
	// no coordination between different handlers.
	AddEventHandlerWithResyncPeriod(handler toolscache.ResourceEventHandler, resyncPeriod time.Duration)
	// AddIndexers adds more indexers to this store.  If you call this after you already have data
	// in the store, the results are undefined.
	AddIndexers(indexers toolscache.Indexers) error
	// HasSynced return true if the informers underlying store has synced.
	HasSynced() bool
}

// ObjectSelector is an alias name of internal.Selector.
type ObjectSelector internal.Selector

// SelectorsByObject associate a client.Object's GVK to a field/label selector.
// There is also `DefaultSelector` to set a global default (which will be overridden by
// a more specific setting here, if any).
type SelectorsByObject map[client.Object]ObjectSelector

// Options are the optional arguments for creating a new InformersMap object.
type Options struct {
	// Scheme is the scheme to use for mapping objects to GroupVersionKinds
	Scheme *runtime.Scheme

	// Mapper is the RESTMapper to use for mapping GroupVersionKinds to Resources
	Mapper meta.RESTMapper

	// Resync is the base frequency the informers are resynced.
	// Defaults to defaultResyncTime.
	// A 10 percent jitter will be added to the Resync period between informers
	// So that all informers will not send list requests simultaneously.
	Resync *time.Duration

	// Namespace restricts the cache's ListWatch to the desired namespace
	// Default watches all namespaces
	Namespace string

	// SelectorsByObject restricts the cache's ListWatch to the desired
	// fields per GVK at the specified object, the map's value must implement
	// Selector [1] using for example a Set [2]
	// [1] https://pkg.go.dev/k8s.io/apimachinery/pkg/fields#Selector
	// [2] https://pkg.go.dev/k8s.io/apimachinery/pkg/fields#Set
	SelectorsByObject SelectorsByObject

	// DefaultSelector will be used as selectors for all object types
	// that do not have a selector in SelectorsByObject defined.
	DefaultSelector ObjectSelector

	// DefaultFieldSelector will be used as a field selector for all object types
	// unless there is already one set in ByObject or DefaultNamespaces.
	DefaultFieldSelector fields.Selector

	// DefaultTransform will be used as transform for all object types
	// unless there is already one set in ByObject or DefaultNamespaces.
	DefaultTransform toolscache.TransformFunc

	// DefaultUnsafeDisableDeepCopy is the default for UnsafeDisableDeepCopy
	// for everything that doesn't specify this.
	//
	// Be very careful with this, when enabled you must DeepCopy any object before mutating it,
	// otherwise you will mutate the object in the cache.
	//
	// This will be used for all object types, unless it is set in ByObject or
	// DefaultNamespaces.
	DefaultUnsafeDisableDeepCopy *bool

	// ByObject restricts the cache's ListWatch to the desired fields per GVK at the specified object.
	// object, this will fall through to Default* settings.
	ByObject map[client.Object]ByObject

	// newInformer allows overriding of NewSharedIndexInformer for testing.
	newInformer *func(toolscache.ListerWatcher, runtime.Object, time.Duration, toolscache.Indexers) toolscache.SharedIndexInformer
}

// ByObject offers more fine-grained control over the cache's ListWatch by object.
type ByObject struct {
	// Namespaces maps a namespace name to cache configs. If set, only the
	// namespaces in this map will be cached.
	//
	// Settings in the map value that are unset will be defaulted.
	// Use an empty value for the specific setting to prevent that.
	//
	// It is possible to have specific Config for just some namespaces
	// but cache all namespaces by using the AllNamespaces const as the map key.
	// This will then include all namespaces that do not have a more specific
	// setting.
	//
	// A nil map allows to default this to the cache's DefaultNamespaces setting.
	// An empty map prevents this and means that all namespaces will be cached.
	//
	// The defaulting follows the following precedence order:
	// 1. ByObject
	// 2. DefaultNamespaces[namespace]
	// 3. Default*
	//
	// This must be unset for cluster-scoped objects.
	Namespaces map[string]Config

	// Label represents a label selector for the object.
	Label labels.Selector

	// Field represents a field selector for the object.
	Field fields.Selector

	// Transform is a transformer function for the object which gets applied
	// when objects of the transformation are about to be committed to the cache.
	//
	// This function is called both for new objects to enter the cache,
	// and for updated objects.
	Transform toolscache.TransformFunc

	// UnsafeDisableDeepCopy indicates not to deep copy objects during get or
	// list objects per GVK at the specified object.
	// Be very careful with this, when enabled you must DeepCopy any object before mutating it,
	// otherwise you will mutate the object in the cache.
	UnsafeDisableDeepCopyByObject DisableDeepCopyByObject
}

var defaultResyncTime = 10 * time.Hour

// New initializes and returns a new Cache.
func New(config *rest.Config, opts Options) (Cache, error) {
	opts, err := defaultOpts(config, opts)
	if err != nil {
		return nil, err
	}
	selectorsByGVK, err := convertToSelectorsByGVK(opts.SelectorsByObject, opts.DefaultSelector, opts.Scheme)
	if err != nil {
		return nil, err
	}
	disableDeepCopyByGVK, err := convertToDisableDeepCopyByGVK(opts.UnsafeDisableDeepCopyByObject, opts.Scheme)
	if err != nil {
		return nil, err
	}
	im := internal.NewInformersMap(config, opts.Scheme, opts.Mapper, *opts.Resync, opts.Namespace, selectorsByGVK, disableDeepCopyByGVK)
	return &informerCache{InformersMap: im}, nil
}

func optionDefaultsToConfig(opts *Options) Config {
	return Config{
		LabelSelector:         opts.DefaultLabelSelector,
		FieldSelector:         opts.DefaultFieldSelector,
		Transform:             opts.DefaultTransform,
		UnsafeDisableDeepCopy: opts.DefaultUnsafeDisableDeepCopy,
	}
}

func byObjectToConfig(byObject ByObject) Config {
	return Config{
		LabelSelector:         byObject.Label,
		FieldSelector:         byObject.Field,
		Transform:             byObject.Transform,
		UnsafeDisableDeepCopy: byObject.UnsafeDisableDeepCopy,
	}
}

type newCacheFunc func(config Config, namespace string) Cache

func newCache(restConfig *rest.Config, opts Options) newCacheFunc {
	return func(config Config, namespace string) Cache {
		return &informerCache{
			scheme: opts.Scheme,
			Informers: internal.NewInformers(restConfig, &internal.InformersOpts{
				HTTPClient:   opts.HTTPClient,
				Scheme:       opts.Scheme,
				Mapper:       opts.Mapper,
				ResyncPeriod: *opts.SyncPeriod,
				Namespace:    namespace,
				Selector: internal.Selector{
					Label: config.LabelSelector,
					Field: config.FieldSelector,
				},
				Transform:             config.Transform,
				UnsafeDisableDeepCopy: pointer.BoolDeref(config.UnsafeDisableDeepCopy, false),
				NewInformer:           opts.newInformer,
			}),
			readerFailOnMissingInformer: opts.ReaderFailOnMissingInformer,
		}
		if options.Mapper == nil {
			options.Mapper = opts.Mapper
		}
		if options.Resync == nil {
			options.Resync = opts.Resync
		}
		if options.Namespace == "" {
			options.Namespace = opts.Namespace
		}
		if opts.Resync == nil {
			opts.Resync = options.Resync
		}

		return New(config, options)
	}
}

func defaultOpts(config *rest.Config, opts Options) (Options, error) {
	// Use the default Kubernetes Scheme if unset
	if opts.Scheme == nil {
		opts.Scheme = scheme.Scheme
	}

	// Construct a new Mapper if unset
	if opts.Mapper == nil {
		var err error
		opts.Mapper, err = apiutil.NewDiscoveryRESTMapper(config, opts.HTTPClient)
		if err != nil {
			log.WithName("setup").Error(err, "Failed to get API Group-Resources")
			return opts, fmt.Errorf("could not create RESTMapper from config")
		}
	}

	for namespace, cfg := range opts.DefaultNamespaces {
		cfg = defaultConfig(cfg, optionDefaultsToConfig(&opts))
		if namespace == metav1.NamespaceAll {
			cfg.FieldSelector = fields.AndSelectors(appendIfNotNil(namespaceAllSelector(maps.Keys(opts.DefaultNamespaces)), cfg.FieldSelector)...)
		}
		opts.DefaultNamespaces[namespace] = cfg
	}

	for obj, byObject := range opts.ByObject {
		isNamespaced, err := apiutil.IsObjectNamespaced(obj, opts.Scheme, opts.Mapper)
		if err != nil {
			return opts, fmt.Errorf("failed to determine if %T is namespaced: %w", obj, err)
		}
		if !isNamespaced && byObject.Namespaces != nil {
			return opts, fmt.Errorf("type %T is not namespaced, but its ByObject.Namespaces setting is not nil", obj)
		}

		// Default the namespace-level configs first, because they need to use the undefaulted type-level config.
		for namespace, config := range byObject.Namespaces {
			// 1. Default from the undefaulted type-level config
			config = defaultConfig(config, byObjectToConfig(byObject))

			// 2. Default from the namespace-level config. This was defaulted from the global default config earlier, but
			//    might not have an entry for the current namespace.
			if defaultNamespaceSettings, hasDefaultNamespace := opts.DefaultNamespaces[namespace]; hasDefaultNamespace {
				config = defaultConfig(config, defaultNamespaceSettings)
			}

			// 3. Default from the global defaults
			config = defaultConfig(config, optionDefaultsToConfig(&opts))

			if namespace == metav1.NamespaceAll {
				config.FieldSelector = fields.AndSelectors(
					appendIfNotNil(
						namespaceAllSelector(maps.Keys(byObject.Namespaces)),
						config.FieldSelector,
					)...,
				)
			}

			byObject.Namespaces[namespace] = config
		}

		defaultedConfig := defaultConfig(byObjectToConfig(byObject), optionDefaultsToConfig(&opts))
		byObject.Label = defaultedConfig.LabelSelector
		byObject.Field = defaultedConfig.FieldSelector
		byObject.Transform = defaultedConfig.Transform
		byObject.UnsafeDisableDeepCopy = defaultedConfig.UnsafeDisableDeepCopy

		if isNamespaced && byObject.Namespaces == nil {
			byObject.Namespaces = opts.DefaultNamespaces
		}

		opts.ByObject[obj] = byObject
	}

	// Default the resync period to 10 hours if unset
	if opts.Resync == nil {
		opts.Resync = &defaultResyncTime
	}
	return opts, nil
}

func convertToSelectorsByGVK(selectorsByObject SelectorsByObject, defaultSelector ObjectSelector, scheme *runtime.Scheme) (internal.SelectorsByGVK, error) {
	selectorsByGVK := internal.SelectorsByGVK{}
	for object, selector := range selectorsByObject {
		gvk, err := apiutil.GVKForObject(object, scheme)
		if err != nil {
			return nil, err
		}
		selectorsByGVK[gvk] = internal.Selector(selector)
	}
	selectorsByGVK[schema.GroupVersionKind{}] = internal.Selector(defaultSelector)
	return selectorsByGVK, nil
}

func namespaceAllSelector(namespaces []string) fields.Selector {
	selectors := make([]fields.Selector, 0, len(namespaces)-1)
	for _, namespace := range namespaces {
		if namespace != metav1.NamespaceAll {
			selectors = append(selectors, fields.OneTermNotEqualSelector("metadata.namespace", namespace))
		}
	}

	return fields.AndSelectors(selectors...)
}

func appendIfNotNil[T comparable](a, b T) []T {
	if b != *new(T) {
		return []T{a, b}
	}
	return []T{a}
}
