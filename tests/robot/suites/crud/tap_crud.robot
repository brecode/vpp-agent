*** Settings ***
Library      OperatingSystem
#Library      RequestsLibrary
#Library      SSHLibrary      timeout=60s
#Library      String

Resource     ../../variables/${VARIABLES}_variables.robot

Resource     ../../libraries/all_libs.robot

Suite Setup       Testsuite Setup
Suite Teardown    Testsuite Teardown
Test Setup        TestSetup
Test Teardown     TestTeardown

*** Variables ***
${VARIABLES}=        common
${ENV}=              common
${NAME_TAP1}=        vpp1_tap1
${NAME_TAP2}=        vpp1_tap2
${MAC_TAP1}=         12:21:21:11:11:11
${MAC_TAP2}=         22:21:21:22:22:22
${IP_TAP1}=          20.20.1.1
${IP_TAP2}=          20.20.1.2
${PREFIX}=           24
${MTU}=              4800





*** Test Cases ***
Configure Environment
    [Tags]    setup
    Configure Environment 1

Show Interfaces Before Setup
    vpp_term: Show Interfaces    agent_vpp_1

Add First TAP Interface
    vpp_ctl: Put TAP Interface With IP    node=agent_vpp_1    name=${NAME_TAP1}    mac=${MAC_TAP1}    ip=${IP_TAP1}    prefix=${PREFIX}    host_if_name=linux_${NAME_TAP1}
    vpp_ctl: Get VPP Interface State    node=agent_vpp_1    interface=${NAME_TAP1}

Add Second TAP Interface
    vpp_ctl: Put TAP Interface With IP    node=agent_vpp_1    name=${NAME_TAP2}    mac=${MAC_TAP2}    ip=${IP_TAP2}    prefix=${PREFIX}    host_if_name=linux_${NAME_TAP2}
    vpp_ctl: Get VPP Interface State    node=agent_vpp_1    interface=${NAME_TAP2}

Check That First TAP Interface Is Still Configured




Add First VXLan Interface
    vxlan: Tunnel Not Exists    node=agent_vpp_1    src=192.168.1.1    dst=192.168.1.2    vni=15
    vpp_ctl: Put VXLan Interface    node=agent_vpp_1    name=vpp1_vxlan1    src=192.168.1.1    dst=192.168.1.2    vni=15
    vxlan: Tunnel Is Created    node=agent_vpp_1    src=192.168.1.1    dst=192.168.1.2    vni=15
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan1    enabled=1    src=192.168.1.1    dst=192.168.1.2    vni=15

Add Second VXLan Interface
    vxlan: Tunnel Not Exists    node=agent_vpp_1    src=192.168.2.1    dst=192.168.2.2    vni=25
    vpp_ctl: Put VXLan Interface    node=agent_vpp_1    name=vpp1_vxlan2    src=192.168.2.1    dst=192.168.2.2    vni=25
    vxlan: Tunnel Is Created    node=agent_vpp_1    src=192.168.2.1    dst=192.168.2.2    vni=25
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan2    enabled=1    src=192.168.2.1    dst=192.168.2.2    vni=25

Check That First VXLan Interface Is Still Configured
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan1    enabled=1    src=192.168.1.1    dst=192.168.1.2    vni=15

Update First VXLan Interface
    vpp_ctl: Put VXLan Interface    node=agent_vpp_1    name=vpp1_vxlan1    src=192.168.1.10    dst=192.168.1.20    vni=150
    vxlan: Tunnel Is Deleted    node=agent_vpp_1    src=192.168.1.1    dst=192.168.1.2    vni=15
    vxlan: Tunnel Is Created    node=agent_vpp_1    src=192.168.1.10    dst=192.168.1.20    vni=150
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan1    enabled=1    src=192.168.1.10    dst=192.168.1.20    vni=150

Check That Second VXLan Interface Is Not Changed
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan2    enabled=1    src=192.168.2.1    dst=192.168.2.2    vni=25

Delete First VXLan Interface
    vpp_ctl: Delete VPP Interface    node=agent_vpp_1    name=vpp1_vxlan1
    vxlan: Tunnel Is Deleted    node=agent_vpp_1    src=192.168.1.10    dst=192.168.1.20    vni=150

Check That Second VXLan Interface Is Still Configured
    vat_term: Check VXLan Interface State    agent_vpp_1    vpp1_vxlan2    enabled=1    src=192.168.2.1    dst=192.168.2.2    vni=25

Show Interfaces And Other Objects After Setup
    vpp_term: Show Interfaces    agent_vpp_1
    vpp_term: Show Interfaces    agent_vpp_2
    Write To Machine    agent_vpp_1_term    show int addr
    Write To Machine    agent_vpp_2_term    show int addr
    Write To Machine    agent_vpp_1_term    show h
    Write To Machine    agent_vpp_2_term    show h
    Write To Machine    agent_vpp_1_term    show br
    Write To Machine    agent_vpp_2_term    show br
    Write To Machine    agent_vpp_1_term    show br 1 detail
    Write To Machine    agent_vpp_2_term    show br 1 detail
    Write To Machine    agent_vpp_1_term    show vxlan tunnel
    Write To Machine    agent_vpp_2_term    show vxlan tunnel
    Write To Machine    agent_vpp_1_term    show err
    Write To Machine    agent_vpp_2_term    show err
    vat_term: Interfaces Dump    agent_vpp_1
    vat_term: Interfaces Dump    agent_vpp_2
    Write To Machine    vpp_agent_ctl    vpp-agent-ctl ${AGENT_VPP_ETCD_CONF_PATH} -ps
    Execute In Container    agent_vpp_1    ip a
    Execute In Container    agent_vpp_2    ip a

*** Keywords ***
TestSetup
    Make Datastore Snapshots    ${TEST_NAME}_test_setup

TestTeardown
    Make Datastore Snapshots    ${TEST_NAME}_test_teardown

