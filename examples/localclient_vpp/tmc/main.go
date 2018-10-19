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

package main

import (
	"context"
	modelTmc "github.com/ligato/vpp-agent/plugins/vpp/model/tmc"
	"log"
	"sync"
	"time"

	"github.com/ligato/cn-infra/agent"
	"github.com/ligato/cn-infra/datasync"
	"github.com/ligato/cn-infra/datasync/kvdbsync/local"
	"github.com/ligato/cn-infra/logging"
	"github.com/ligato/cn-infra/logging/logrus"
	"github.com/ligato/vpp-agent/clientv1/vpp/localclient"
	"github.com/ligato/vpp-agent/plugins/vpp"
	vpp_intf "github.com/ligato/vpp-agent/plugins/vpp/model/interfaces"
	"github.com/namsral/flag"
)

var (
	timeout = flag.Int("timeout", 20, "Timeout between applying of global and DNAT configuration in seconds")
)

/* Vpp-agent Init and Close*/

// Start Agent plugins selected for this example.
func main() {
	//Init close channel to stop the example.
	exampleFinished := make(chan struct{}, 1)
	// Prepare all the dependencies for example plugin
	watcher := datasync.KVProtoWatchers{
		local.Get(),
	}
	vppPlugin := vpp.NewPlugin(vpp.UseDeps(func(deps *vpp.Deps) {
		deps.Watcher = watcher
	}))

	// Inject dependencies to example plugin
	ep := &TmcExamplePlugin{}
	ep.Deps.Log = logging.DefaultLogger
	ep.Deps.VPP = vppPlugin

	// Start Agent
	a := agent.NewAgent(
		agent.AllPlugins(ep),
		agent.QuitOnClose(exampleFinished),
	)
	if err := a.Run(); err != nil {
		log.Fatal()
	}

	go closeExample("localhost example finished", exampleFinished)
}

// Stop the agent with desired info message.
func closeExample(message string, exampleFinished chan struct{}) {
	time.Sleep(time.Duration(*timeout+5) * time.Second)
	logrus.DefaultLogger().Info(message)
	close(exampleFinished)
}

// TmcExamplePlugin uses localclient to transport example global NAT and DNAT and af-packet
// configuration to NAT VPP plugin
type TmcExamplePlugin struct {
	Deps

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// Deps is example plugin dependencies.
type Deps struct {
	Log logging.Logger
	VPP *vpp.Plugin
}

// PluginName represents name of plugin.
const PluginName = "tmc-example"

// Init initializes example plugin.
func (plugin *TmcExamplePlugin) Init() error {
	// Logger
	plugin.Log = logrus.DefaultLogger()
	plugin.Log.SetLevel(logging.DebugLevel)
	plugin.Log.Info("Initializing NAT44 example")

	// Flags
	flag.Parse()
	plugin.Log.Infof("Timeout between configuring NAT global and DNAT set to %d", *timeout)

	// Apply initial VPP configuration.
	plugin.putGlobalConfig()

	// Schedule reconfiguration.
	var ctx context.Context
	ctx, plugin.cancel = context.WithCancel(context.Background())
	plugin.wg.Add(1)
	go plugin.applyTmcConfig(ctx, *timeout)

	plugin.Log.Info("NAT example initialization done")
	return nil
}

// Close cleans up the resources.
func (plugin *TmcExamplePlugin) Close() error {
	plugin.cancel()
	plugin.wg.Wait()

	logrus.DefaultLogger().Info("Closed NAT example plugin")
	return nil
}

// String returns plugin name
func (plugin *TmcExamplePlugin) String() string {
	return PluginName
}

// ConfigureZ Global config
func (plugin *TmcExamplePlugin) putGlobalConfig() {
	plugin.Log.Infof("Applying  global configuration")
	err := localclient.DataResyncRequest(PluginName).
		Interface(interface1()).
		Interface(interface2()).
		Interface(interface3()).
		Send().ReceiveReply()
	if err != nil {
		plugin.Log.Errorf("Global configuration failed: %v", err)
	} else {
		plugin.Log.Info("Global configuration successful")
	}
}

// Configure DNAT
func (plugin *TmcExamplePlugin) applyTmcConfig(ctx context.Context, timeout int) {
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		plugin.Log.Infof("Applying TMC configuration")
		err := localclient.DataChangeRequest(PluginName).
			Put().TmcConfig(tmcConfig()).
			Send().ReceiveReply()
		if err != nil {
			plugin.Log.Errorf("TMC configuration failed: %v", err)
		} else {
			plugin.Log.Info("TMC configuration successful")
		}
	case <-ctx.Done():
		// Cancel the scheduled DNAT configuration.
		plugin.Log.Info("TMC configuration canceled")
	}
	plugin.wg.Done()
}

/* Example Data */

func interface1() *vpp_intf.Interfaces_Interface {
	return &vpp_intf.Interfaces_Interface{
		Name:    "memif1",
		Type:    vpp_intf.InterfaceType_MEMORY_INTERFACE,
		Enabled: true,
		Mtu:     1478,
		IpAddresses: []string{
			"172.125.40.1/24",
		},
		Memif: &vpp_intf.Interfaces_Interface_Memif{
			Id:             1,
			Secret:         "secret1",
			Master:         false,
			SocketFilename: "/tmp/memif1.sock",
		},
	}
}

func interface2() *vpp_intf.Interfaces_Interface {
	return &vpp_intf.Interfaces_Interface{
		Name:    "memif2",
		Type:    vpp_intf.InterfaceType_MEMORY_INTERFACE,
		Enabled: true,
		Mtu:     1478,
		IpAddresses: []string{
			"192.47.21.1/24",
		},
		Memif: &vpp_intf.Interfaces_Interface_Memif{
			Id:             2,
			Secret:         "secret2",
			Master:         false,
			SocketFilename: "/tmp/memif1.sock",
		},
	}
}

func interface3() *vpp_intf.Interfaces_Interface {
	return &vpp_intf.Interfaces_Interface{
		Name:    "memif3",
		Type:    vpp_intf.InterfaceType_MEMORY_INTERFACE,
		Enabled: true,
		Mtu:     1478,
		IpAddresses: []string{
			"94.18.21.1/24",
		},
		Memif: &vpp_intf.Interfaces_Interface_Memif{
			Id:             3,
			Secret:         "secret3",
			Master:         false,
			SocketFilename: "/tmp/memif1.sock",
		},
	}
}

func tmcConfig() *modelTmc.TmcConfig {
	return &modelTmc.TmcConfig{
		ConfigName:     "config1",
		TcpMssClamping: 1,
		InterfaceName:  "memif1",
		MssValue:       1450,
	}
}
