package plugin_config

import (
	"path/filepath"
	"sync"

	"github.com/cloudfoundry/cli/cf/configuration"
	"github.com/cloudfoundry/cli/cf/configuration/config_helpers"
)

type PluginConfiguration interface {
	Plugins() map[string]string
	SetPlugin(string, string)
	GetPluginPath() string
}

type PluginConfig struct {
	mutex      *sync.RWMutex
	initOnce   *sync.Once
	persistor  configuration.Persistor
	onError    func(error)
	data       *PluginData
	pluginPath string
}

func NewPluginConfig(errorHandler func(error)) *PluginConfig {
	pluginPath := filepath.Join(config_helpers.PluginRepoDir(), ".cf", "plugins")
	return &PluginConfig{
		data:       NewData(),
		mutex:      new(sync.RWMutex),
		initOnce:   new(sync.Once),
		persistor:  configuration.NewDiskPersistor(filepath.Join(pluginPath, "config.json")),
		onError:    errorHandler,
		pluginPath: pluginPath,
	}
}

/* getter methods */
func (c *PluginConfig) GetPluginPath() string {
	return c.pluginPath
}

func (c *PluginConfig) Plugins() map[string]string {
	c.read()
	return c.data.Plugins
}

/* setter methods */
func (c *PluginConfig) SetPlugin(name, location string) {
	if c.data.Plugins == nil {
		c.data.Plugins = make(map[string]string)
	}
	c.write(func() {
		c.data.Plugins[name] = location
	})
}

/* Functions that handel locking */
func (c *PluginConfig) init() {
	c.initOnce.Do(func() {
		err := c.persistor.Load(c.data)
		if err != nil {
			c.onError(err)
		}
	})
}

func (c *PluginConfig) read() {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	c.init()
}

func (c *PluginConfig) write(cb func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.init()

	cb()

	err := c.persistor.Save(c.data)
	if err != nil {
		c.onError(err)
	}
}

// CLOSERS
func (c *PluginConfig) Close() {
	c.read()
	// perform a read to ensure write lock has been cleared
}
