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

package vppcalls

import (
	"fmt"
	"github.com/ligato/vpp-agent/plugins/vpp/binapi/tmc"
)

// TmcConfig represents TMC config entry
type TmcConfig struct {
	MssValue  uint16
	IfaceIdx  uint32
	IsEnabled uint8
}

func (h *TmcVppHandler) ModifyTmcConfig(ifIdx uint32, mssValue uint16, isEnabled uint8) error {
	// prepare the message
	req := &tmc.TmcEnableDisable{
		IsEnable:  isEnabled,
		SwIfIndex: ifIdx,
		Mss:       mssValue,
	}

	reply := &tmc.TmcEnableDisableReply{}

	if err := h.callsChannel.SendRequest(req).ReceiveReply(reply); err != nil {
		return err
	} else if reply.Retval != 0 {
		return fmt.Errorf("%s returned %d", reply.GetMessageName(), reply.Retval)
	}

	return nil

}
