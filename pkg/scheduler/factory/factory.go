package factory

import (
	"fmt"
	"regexp"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	"k8s.io/kubernetes/pkg/scheduler/algorithm"
	"k8s.io/kubernetes/pkg/scheduler/factory"
)

var (
	schedulerFactoryMutex sync.Mutex

	// maps that hold registered algorithm types
	fitPredicateMap        = make(map[string]factory.FitPredicateFactory)
	mandatoryFitPredicates = sets.NewString()
	priorityFunctionMap    = make(map[string]factory.PriorityConfigFactory)
	algorithmProviderMap   = make(map[string]factory.AlgorithmProviderConfig)

	// Registered metadata producers
	priorityMetadataProducer  factory.PriorityMetadataProducerFactory
	predicateMetadataProducer factory.PredicateMetadataProducerFactory
)

type Config struct {
	NodeLister        algorithm.NodeLister
	ServiceLister     algorithm.ServiceLister
	ControllerLister  algorithm.ControllerLister
	ReplicaSetLister  algorithm.ReplicaSetLister
	StatefulSetLister algorithm.StatefulSetLister
}

func (c *Config) GetPluginArgs() (*factory.PluginFactoryArgs, error) {
	return &factory.PluginFactoryArgs{
		NodeLister:        c.NodeLister,
		ServiceLister:     c.ServiceLister,
		ControllerLister:  c.ControllerLister,
		ReplicaSetLister:  c.ReplicaSetLister,
		StatefulSetLister: c.StatefulSetLister,
	}, nil
}

func (c *Config) GetPredicates(predicateKeys sets.String) (map[string]algorithm.FitPredicate, error) {
	pluginArgs, err := c.GetPluginArgs()
	if err != nil {
		return nil, err
	}

	return GetFitPredicateFunctions(predicateKeys, *pluginArgs)
}

func GetFitPredicateFunctions(names sets.String, args factory.PluginFactoryArgs) (map[string]algorithm.FitPredicate, error) {
	predicates := map[string]algorithm.FitPredicate{}
	for _, name := range names.List() {
		factory, ok := fitPredicateMap[name]
		if !ok {
			return nil, fmt.Errorf("Invalid predicate name %q specified - no corresponding function found", name)
		}
		predicates[name] = factory(args)
	}

	// Always include mandatory fit predicates.
	for name := range mandatoryFitPredicates {
		if factory, found := fitPredicateMap[name]; found {
			predicates[name] = factory(args)
		}
	}

	return predicates, nil
}

func (c *Config) GetPriorityFunctionConfigs(names sets.String) ([]algorithm.PriorityConfig, error) {
	pluginArgs, err := c.GetPluginArgs()
	if err != nil {
		return nil, err
	}

	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	configs := []algorithm.PriorityConfig{}
	for _, name := range names.List() {
		factory, ok := priorityFunctionMap[name]
		if !ok {
			return nil, fmt.Errorf("Invalid priority name %s specified - no corresponding function found", name)
		}
		if factory.Function != nil {
			configs = append(configs, algorithm.PriorityConfig{
				Name:     name,
				Function: factory.Function(*pluginArgs),
				Weight:   factory.Weight,
			})
		} else {
			mapFunction, reduceFunction := factory.MapReduceFunction(*pluginArgs)
			configs = append(configs, algorithm.PriorityConfig{
				Name:   name,
				Map:    mapFunction,
				Reduce: reduceFunction,
				Weight: factory.Weight,
			})
		}
	}
	return configs, nil
}

func RegisterFitPredicate(name string, predicate algorithm.FitPredicate) string {
	return RegisterFitPredicateFactory(name, func(factory.PluginFactoryArgs) algorithm.FitPredicate { return predicate })
}

func RegisterFitPredicateFactory(name string, predicateFactory factory.FitPredicateFactory) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	fitPredicateMap[name] = predicateFactory
	return name
}

var validName = regexp.MustCompile("^[a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])$")

func validateAlgorithmNameOrDie(name string) {
	if !validName.MatchString(name) {
		klog.Fatalf("Algorithm name %v does not match the name validation regexp \"%v\".", name, validName)
	}
}

// RegisterPredicateMetadataProducerFactory registers a PredicateMetadataProducerFactory.
func RegisterPredicateMetadataProducerFactory(factory factory.PredicateMetadataProducerFactory) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	predicateMetadataProducer = factory
}

func RegisterPriorityFunction2(
	name string,
	mapFunction algorithm.PriorityMapFunction,
	reduceFunction algorithm.PriorityReduceFunction,
	weight int) string {
	return RegisterPriorityConfigFactory(name, factory.PriorityConfigFactory{
		MapReduceFunction: func(factory.PluginFactoryArgs) (algorithm.PriorityMapFunction, algorithm.PriorityReduceFunction) {
			return mapFunction, reduceFunction
		},
		Weight: weight,
	})
}

// RegisterPriorityConfigFactory registers a priority config factory with its name.
func RegisterPriorityConfigFactory(name string, pcf factory.PriorityConfigFactory) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	priorityFunctionMap[name] = pcf
	return name
}

// RegisterPriorityMetadataProducerFactory registers a PriorityMetadataProducerFactory.
func RegisterPriorityMetadataProducerFactory(factory factory.PriorityMetadataProducerFactory) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	priorityMetadataProducer = factory
}

// RemoveFitPredicate removes a fit predicate from factory.
func RemoveFitPredicate(name string) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	validateAlgorithmNameOrDie(name)
	delete(fitPredicateMap, name)
	mandatoryFitPredicates.Delete(name)
}

// RemovePredicateKeyFromAlgorithmProviderMap removes a fit predicate key from all algorithmProviders which in algorithmProviderMap.
func RemovePredicateKeyFromAlgorithmProviderMap(key string) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	for _, provider := range algorithmProviderMap {
		provider.FitPredicateKeys.Delete(key)
	}
	return
}

// InsertPredicateKeyToAlgorithmProviderMap insert a fit predicate key to all algorithmProviders which in algorithmProviderMap.
func InsertPredicateKeyToAlgorithmProviderMap(key string) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	for _, provider := range algorithmProviderMap {
		provider.FitPredicateKeys.Insert(key)
	}
	return
}

// RegisterAlgorithmProvider registers a new algorithm provider with the algorithm registry. This should
// be called from the init function in a provider plugin.
func RegisterAlgorithmProvider(name string, predicateKeys, priorityKeys sets.String) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	algorithmProviderMap[name] = factory.AlgorithmProviderConfig{
		FitPredicateKeys:     predicateKeys,
		PriorityFunctionKeys: priorityKeys,
	}
	return name
}

// RegisterMandatoryFitPredicate registers a fit predicate with the algorithm registry, the predicate is used by
// kubelet, DaemonSet; it is always included in configuration. Returns the name with which the predicate was
// registered.
func RegisterMandatoryFitPredicate(name string, predicate algorithm.FitPredicate) string {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()
	validateAlgorithmNameOrDie(name)
	fitPredicateMap[name] = func(factory.PluginFactoryArgs) algorithm.FitPredicate { return predicate }
	mandatoryFitPredicates.Insert(name)
	return name
}

// InsertPriorityKeyToAlgorithmProviderMap inserts a priority function to all algorithmProviders which are in algorithmProviderMap.
func InsertPriorityKeyToAlgorithmProviderMap(key string) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	for _, provider := range algorithmProviderMap {
		provider.PriorityFunctionKeys.Insert(key)
	}
	return
}

// GetAlgorithmProvider should not be used to modify providers. It is publicly visible for testing.
func GetAlgorithmProvider(name string) (*factory.AlgorithmProviderConfig, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	provider, ok := algorithmProviderMap[name]
	if !ok {
		return nil, fmt.Errorf("plugin %q has not been registered", name)
	}

	return &provider, nil
}
func (c *Config) GetPredicateMetadataProducer() (algorithm.PredicateMetadataProducer, error) {
	pluginArgs, err := c.GetPluginArgs()
	if err != nil {
		return nil, err
	}
	return getPredicateMetadataProducer(*pluginArgs)
}

func getPredicateMetadataProducer(args factory.PluginFactoryArgs) (algorithm.PredicateMetadataProducer, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	if predicateMetadataProducer == nil {
		return algorithm.EmptyPredicateMetadataProducer, nil
	}
	return predicateMetadataProducer(args), nil
}

func (c *Config) GetPriorityMetadataProducer() (algorithm.PriorityMetadataProducer, error) {
	pluginArgs, err := c.GetPluginArgs()
	if err != nil {
		return nil, err
	}

	return getPriorityMetadataProducer(*pluginArgs)
}

func getPriorityMetadataProducer(args factory.PluginFactoryArgs) (algorithm.PriorityMetadataProducer, error) {
	schedulerFactoryMutex.Lock()
	defer schedulerFactoryMutex.Unlock()

	if priorityMetadataProducer == nil {
		return algorithm.EmptyPriorityMetadataProducer, nil
	}
	return priorityMetadataProducer(args), nil
}
