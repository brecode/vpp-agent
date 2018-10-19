// Copyright (c) 2017 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ifplugin

import (
	"fmt"

	govppapi "git.fd.io/govpp.git/api"
	"github.com/go-errors/errors"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/utils/safeclose"
	"github.com/ligato/vpp-agent/idxvpp"
	"github.com/ligato/vpp-agent/idxvpp/nametoidx"
	"github.com/ligato/vpp-agent/plugins/govppmux"
	"github.com/ligato/vpp-agent/plugins/vpp/ifplugin/ifaceidx"
	"github.com/ligato/vpp-agent/plugins/vpp/ifplugin/vppcalls"
	modelTmc "github.com/ligato/vpp-agent/plugins/vpp/model/tmc"
)

// TmcConfigurator runs in the background in its own goroutine where it watches for any changes
// in the configuration of interfaces as modelled by the proto file "../model/tmc/tmc.proto"
// and stored in ETCD under the key "vpp/config/v1/tmc/config/".
type TmcConfigurator struct {
	log logging.Logger
	// Indexes
	ifIndexes        ifaceidx.SwIfIndex
	allIndexes       idxvpp.NameToIdxRW
	allIndexesSeq    uint32
	unstoredIndexes  idxvpp.NameToIdxRW
	unstoredIndexSeq uint32
	// VPP
	vppChan govppapi.Channel
	// VPP API handler
	tmcHandler vppcalls.TmcVppAPI
}

// IndexExistsFor returns true if there is a mapping entry for provided name
func (c *TmcConfigurator) IndexExistsFor(name string) bool {
	_, _, found := c.allIndexes.LookupIdx(name)
	return found
}

// UnstoredIndexExistsFor returns true if there is a mapping entry for provided name
func (c *TmcConfigurator) UnstoredIndexExistsFor(name string) bool {
	_, _, found := c.unstoredIndexes.LookupIdx(name)
	return found
}

// Init initializes TMC configurator
func (c *TmcConfigurator) Init(logger logging.PluginLogger, goVppMux govppmux.API, ifIndexes ifaceidx.SwIfIndex) (err error) {
	// Init logger
	c.log = logger.NewLogger("tmc-conf")

	// Init VPP API channel
	if c.vppChan, err = goVppMux.NewAPIChannel(); err != nil {
		return errors.Errorf("failed to create API channel: %v", err)
	}

	// Init indexes
	c.ifIndexes = ifIndexes
	c.allIndexes = nametoidx.NewNameToIdx(c.log, "tmc-all-indexes", nil)
	c.unstoredIndexes = nametoidx.NewNameToIdx(c.log, "tmc-unstored-indexes", nil)
	c.allIndexesSeq, c.unstoredIndexSeq = 1, 1

	// VPP API handler
	c.tmcHandler = vppcalls.NewTmcVppHandler(c.vppChan, c.ifIndexes, c.log)

	c.log.Info("TMC configurator initialized")

	return nil
}

// clearMapping prepares all in-memory-mappings and other cache fields. All previous cached entries are removed.
func (c *TmcConfigurator) clearMapping() {
	c.allIndexes.Clear()
	c.unstoredIndexes.Clear()
}

// ResolveDeletedInterface resolves when interface is deleted. If there exist a config for this interface
// the config will be deleted also.
func (c *TmcConfigurator) ResolveDeletedInterface(interfaceName string) error {
	if config := c.configFromIndex(interfaceName, true); config != nil {
		if err := c.Delete(config); err != nil {
			return err
		}
	}
	return nil
}

// ResolveCreatedInterface will check configs and if there is one waiting for interfaces it will be written
// into VPP.
func (c *TmcConfigurator) ResolveCreatedInterface(interfaceName string) error {
	if config := c.configFromIndex(interfaceName, false); config != nil {
		if err := c.Add(config); err == nil {
			c.unstoredIndexes.UnregisterName(TmcIdentifier(interfaceName))
			c.log.Debugf("tmc config %s unregistered", config.ConfigName)
		} else {
			return err
		}
	}
	return nil
}

// Add create a new tmc config.
func (c *TmcConfigurator) Add(config *modelTmc.TmcConfig) error {
	// Check tmc data
	tmcConfig, doVPPCall, err := c.checkTmc(config, c.ifIndexes)
	if err != nil {
		return err
	}

	if !doVPPCall {
		c.log.Debugf("There is no interface for config: %v. Waiting for interface.", config.ConfigName)
		c.indexTmcConfig(config, true)
	} else {
		// Create and register new tmc
		if err := c.tmcHandler.ModifyTmcConfig(tmcConfig.IfaceIdx, tmcConfig.MssValue, tmcConfig.IsEnabled); err != nil {
			return errors.Errorf("failed to add tmc config %s: %v", config.ConfigName, err)
		}
		c.indexTmcConfig(config, false)

		c.log.Infof("tmc config %s configured", config.ConfigName)
	}

	return nil
}

// Delete removes tmc config.
func (c *TmcConfigurator) Delete(config *modelTmc.TmcConfig) error {
	// Check tmc data
	tmcConfig, _, err := c.checkTmc(config, c.ifIndexes)
	if err != nil {
		return err
	}

	if withoutIf, _ := c.removeConfigFromIndex(config.InterfaceName); withoutIf {
		c.log.Debug("tmc config was not stored into VPP, removed only from indexes.")
		return nil
	}

	// Remove config
	if err := c.tmcHandler.ModifyTmcConfig(tmcConfig.IfaceIdx, tmcConfig.MssValue, tmcConfig.IsEnabled); err != nil {
		return errors.Errorf("failed to delete tmc config %s: %v", config.ConfigName, err)
	}

	c.log.Infof("TMC config %s removed", config.ConfigName)

	return nil
}

// Modify configured config.
func (c *TmcConfigurator) Modify(oldConfig *modelTmc.TmcConfig, newConfig *modelTmc.TmcConfig) error {
	if oldConfig == nil {
		return errors.Errorf("failed to modify tmc config, provided old value is nil")
	}

	if newConfig == nil {
		return errors.Errorf("failed to modify tmc config, provided new value is nil")
	}

	if err := c.Delete(oldConfig); err != nil {
		return err
	}

	if err := c.Add(newConfig); err != nil {
		return err
	}

	c.log.Infof("TMC config %s modified", newConfig.ConfigName)

	return nil
}

// Close GOVPP channel.
func (c *TmcConfigurator) Close() error {
	if err := safeclose.Close(c.vppChan); err != nil {
		return c.LogError(errors.Errorf("failed to safeclose tmc configurator: %v", err))
	}
	return nil
}

// checkTmc will check the config raw data and change it to internal data structure.
// In case the config contains a interface that doesn't exist yet, config is stored into index map.
func (c *TmcConfigurator) checkTmc(tmcInput *modelTmc.TmcConfig, index ifaceidx.SwIfIndex) (tmcConfig *vppcalls.TmcConfig, doVPPCall bool, err error) {
	c.log.Debugf("Checking tmc config: %+v", tmcInput)

	if tmcInput == nil {
		return nil, false, errors.Errorf("failed to add tmc config, input is empty")
	}
	if tmcInput.InterfaceName == "" {
		return nil, false, errors.Errorf("failed to add tmc config %s, no interface provided",
			tmcInput.ConfigName)
	}

	ifName := tmcInput.InterfaceName
	ifIndex, _, exists := index.LookupIdx(ifName)
	if exists {
		doVPPCall = true
	}

	tmcConfig = &vppcalls.TmcConfig{
		MssValue:  uint16(tmcInput.MssValue),
		IfaceIdx:  ifIndex,
		IsEnabled: uint8(tmcInput.TcpMssClamping),
	}

	return
}

func (c *TmcConfigurator) indexTmcConfig(config *modelTmc.TmcConfig, withoutIface bool) {
	idx := TmcIdentifier(config.InterfaceName)
	if withoutIface {
		c.unstoredIndexes.RegisterName(idx, c.unstoredIndexSeq, config)
		c.unstoredIndexSeq++
		c.log.Debugf("tmc config %s cached to unstored", config.ConfigName)
	}
	c.allIndexes.RegisterName(idx, c.allIndexesSeq, config)
	c.allIndexesSeq++
	c.log.Debugf("tmc config %s registered to all", config.ConfigName)
}

func (c *TmcConfigurator) removeConfigFromIndex(iface string) (withoutIface bool, config *modelTmc.TmcConfig) {
	idx := TmcIdentifier(iface)

	// Removing config from main index
	_, configIface, exists := c.allIndexes.LookupIdx(idx)
	if exists {
		c.allIndexes.UnregisterName(idx)
		c.log.Debugf("tmc config %d unregistered from all", idx)
		tmcConfig, ok := configIface.(*modelTmc.TmcConfig)
		if ok {
			config = tmcConfig
		}
	}

	// Removing config from not stored config index
	_, _, existsWithout := c.unstoredIndexes.LookupIdx(idx)
	if existsWithout {
		withoutIface = true
		c.unstoredIndexes.UnregisterName(idx)
		c.log.Debugf("tmc config %s unregistered from unstored", config.ConfigName)
	}

	return
}

func (c *TmcConfigurator) configFromIndex(iface string, fromAllConfigs bool) (config *modelTmc.TmcConfig) {
	idx := TmcIdentifier(iface)

	var configIface interface{}
	var exists bool

	if !fromAllConfigs {
		_, configIface, exists = c.unstoredIndexes.LookupIdx(idx)
	} else {
		_, configIface, exists = c.allIndexes.LookupIdx(idx)
	}
	if exists {
		tmcConfig, ok := configIface.(*modelTmc.TmcConfig)
		if ok {
			config = tmcConfig
		}
	}

	return
}

// TmcIdentifier creates unique identifier which serves as a name in name to index mapping
func TmcIdentifier(iface string) string {
	return fmt.Sprintf("tmc-iface-%v", iface)
}

// LogError prints error if not nil, including stack trace. The same value is also returned, so it can be easily propagated further
func (c *TmcConfigurator) LogError(err error) error {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *errors.Error:
		c.log.WithField("logger", c.log).Errorf(string(err.Error() + "\n" + string(err.(*errors.Error).Stack())))
	default:
		c.log.Error(err)
	}
	return err
}
