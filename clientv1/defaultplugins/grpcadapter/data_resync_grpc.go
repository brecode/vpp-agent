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

package grpcadapter

import (
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/ligato/vpp-agent/clientv1/defaultplugins"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/acl"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/bfd"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/interfaces"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/ipsec"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/l2"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/l3"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/l4"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/nat"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/rpc"
	"github.com/ligato/vpp-agent/plugins/defaultplugins/common/model/stn"
	"golang.org/x/net/context"
)

// NewDataResyncDSL is a constructor.
func NewDataResyncDSL(client rpc.ResyncConfigServiceClient) *DataResyncDSL {
	return &DataResyncDSL{client, make(map[string]proto.Message)}
}

// DataResyncDSL is used to conveniently assign all the data that are needed for the RESYNC.
// This is implementation of Domain Specific Language (DSL) for data RESYNC of the VPP configuration.
type DataResyncDSL struct {
	client rpc.ResyncConfigServiceClient
	put    map[string]proto.Message
}

// Interface adds Bridge Domain to the RESYNC request.
func (dsl *DataResyncDSL) Interface(val *interfaces.Interfaces_Interface) defaultplugins.DataResyncDSL {
	dsl.put[val.Name] = val

	return dsl
}

// BfdSession adds BFD session to the RESYNC request.
func (dsl *DataResyncDSL) BfdSession(val *bfd.SingleHopBFD_Session) defaultplugins.DataResyncDSL {
	dsl.put[val.Interface] = val

	return dsl
}

// BfdAuthKeys adds BFD key to the RESYNC request.
func (dsl *DataResyncDSL) BfdAuthKeys(val *bfd.SingleHopBFD_Key) defaultplugins.DataResyncDSL {
	dsl.put[strconv.Itoa(int(val.Id))] = val

	return dsl
}

// BfdEchoFunction adds BFD echo function to the RESYNC request.
func (dsl *DataResyncDSL) BfdEchoFunction(val *bfd.SingleHopBFD_EchoFunction) defaultplugins.DataResyncDSL {
	dsl.put[val.EchoSourceInterface] = val

	return dsl
}

// BD adds Bridge Domain to the RESYNC request.
func (dsl *DataResyncDSL) BD(val *l2.BridgeDomains_BridgeDomain) defaultplugins.DataResyncDSL {
	dsl.put[val.Name] = val

	return dsl
}

// BDFIB adds Bridge Domain to the RESYNC request.
func (dsl *DataResyncDSL) BDFIB(val *l2.FibTableEntries_FibTableEntry) defaultplugins.DataResyncDSL {
	dsl.put[l2.FibKey(val.BridgeDomain, val.PhysAddress)] = val

	return dsl
}

// XConnect adds Cross Connect to the RESYNC request.
func (dsl *DataResyncDSL) XConnect(val *l2.XConnectPairs_XConnectPair) defaultplugins.DataResyncDSL {
	dsl.put[val.ReceiveInterface] = val

	return dsl
}

// StaticRoute adds L3 Static Route to the RESYNC request.
func (dsl *DataResyncDSL) StaticRoute(val *l3.StaticRoutes_Route) defaultplugins.DataResyncDSL {
	dsl.put[l3.RouteKey(val.VrfId, val.DstIpAddr, val.NextHopAddr)] = val

	return dsl
}

// ACL adds Access Control List to the RESYNC request.
func (dsl *DataResyncDSL) ACL(val *acl.AccessLists_Acl) defaultplugins.DataResyncDSL {
	dsl.put[val.AclName] = val

	return dsl
}

// L4Features adds L4Features to the RESYNC request.
func (dsl *DataResyncDSL) L4Features(val *l4.L4Features) defaultplugins.DataResyncDSL {
	dsl.put[strconv.FormatBool(val.Enabled)] = val

	return dsl
}

// AppNamespace adds Application Namespace to the RESYNC request.
func (dsl *DataResyncDSL) AppNamespace(val *l4.AppNamespaces_AppNamespace) defaultplugins.DataResyncDSL {
	dsl.put[val.NamespaceId] = val

	return dsl
}

// Arp adds VPP L3 ARP to the RESYNC request.
func (dsl *DataResyncDSL) Arp(arp *l3.ArpTable_ArpTableEntry) defaultplugins.DataResyncDSL {
	dsl.put[l3.ArpEntryKey(arp.Interface, arp.IpAddress)] = arp
	return dsl
}

// ProxyArpInterfaces adds L3 proxy ARP interfaces to the RESYNC request.
func (dsl *DataResyncDSL) ProxyArpInterfaces(val *l3.ProxyArpInterfaces_InterfaceList) defaultplugins.DataResyncDSL {
	dsl.put[val.Label] = val
	return dsl
}

// ProxyArpRanges adds L3 proxy ARP ranges to the RESYNC request.
func (dsl *DataResyncDSL) ProxyArpRanges(val *l3.ProxyArpRanges_RangeList) defaultplugins.DataResyncDSL {
	dsl.put[val.Label] = val
	return dsl
}

// StnRule adds Stn rule to the RESYNC request.
func (dsl *DataResyncDSL) StnRule(val *stn.StnRule) defaultplugins.DataResyncDSL {
	dsl.put[val.RuleName] = val
	return dsl
}

// NAT44Global adds a request to RESYNC global configuration for NAT44
func (dsl *DataResyncDSL) NAT44Global(nat44 *nat.Nat44Global) defaultplugins.DataResyncDSL {
	dsl.put["global"] = nat44
	return dsl
}

// NAT44DNat adds a request to RESYNC a new DNAT configuration
func (dsl *DataResyncDSL) NAT44DNat(nat44 *nat.Nat44DNat_DNatConfig) defaultplugins.DataResyncDSL {
	dsl.put[nat.DNatKey(nat44.Label)] = nat44
	return dsl
}

// IPSecSA adds request to create a new Security Association
func (dsl *DataResyncDSL) IPSecSA(sa *ipsec.SecurityAssociations_SA) defaultplugins.DataResyncDSL {
	dsl.put[ipsec.SAKey(sa.Name)] = sa
	return dsl
}

// IPSecSPD adds request to create a new Security Policy Database
func (dsl *DataResyncDSL) IPSecSPD(spd *ipsec.SecurityPolicyDatabases_SPD) defaultplugins.DataResyncDSL {
	dsl.put[ipsec.SPDKey(spd.Name)] = spd
	return dsl
}

// Send propagates the request to the plugins. It deletes obsolete keys if listKeys() function is not null.
// The listkeys() function is used to list all current keys.
func (dsl *DataResyncDSL) Send() defaultplugins.Reply {

	resyncReq := &rpc.ResyncConfigRequest{}

	for _, val := range dsl.put {
		switch typed := val.(type) {
		case *interfaces.Interfaces_Interface:
			if resyncReq.Interfaces == nil {
				resyncReq.Interfaces = &interfaces.Interfaces{}
			}
			resyncReq.Interfaces.Interface = append(resyncReq.Interfaces.Interface, typed)
		case *l2.BridgeDomains_BridgeDomain:
			if resyncReq.BDs == nil {
				resyncReq.BDs = &l2.BridgeDomains{}
			}
			resyncReq.BDs.BridgeDomains = append(resyncReq.BDs.BridgeDomains, typed)
		case *l2.XConnectPairs_XConnectPair:
			if resyncReq.XCons == nil {
				resyncReq.XCons = &l2.XConnectPairs{}
			}
			resyncReq.XCons.XConnectPairs = append(resyncReq.XCons.XConnectPairs, typed)
		case *l3.StaticRoutes_Route:
			if resyncReq.StaticRoutes == nil {
				resyncReq.StaticRoutes = &l3.StaticRoutes{}
			}
			resyncReq.StaticRoutes.Route = append(resyncReq.StaticRoutes.Route, typed)
		case *acl.AccessLists_Acl:
			if resyncReq.ACLs == nil {
				resyncReq.ACLs = &acl.AccessLists{}
			}
			resyncReq.ACLs.Acl = append(resyncReq.ACLs.Acl, typed)
		}
	}

	_, err := dsl.client.ResyncConfig(context.Background(), resyncReq)

	return &Reply{err: err}
}
