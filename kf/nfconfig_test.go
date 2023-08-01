// Copyright Contributors to the L3AF Project.
// SPDX-License-Identifier: Apache-2.0

package kf

import (
	"container/list"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/l3af-project/l3afd/config"
	"github.com/l3af-project/l3afd/mocks"
	"github.com/l3af-project/l3afd/models"

	"github.com/rs/zerolog/log"
)

var (
	machineHostname string
	hostInterfaces  map[string]bool
	pMon            *pCheck
	mMon            *kfMetrics
	//	val             []byte
	valVerChange    *models.BPFPrograms
	valStatusChange *models.BPFPrograms
	ingressXDPBpfs  map[string]*list.List
	ingressTCBpfs   map[string]*list.List
	egressTCBpfs    map[string]*list.List
	ifaceName       string
	seqID           int
	bpfProgs        *models.BPFPrograms
)

func setupDBTest() {
	machineHostname, _ = os.Hostname()
	hostInterfaces = make(map[string]bool)
	hostInterfaces["fakeif0"] = true
	pMon = NewpCheck(3, true, 10)
	mMon = NewpKFMetrics(true, 30)

	ingressXDPBpfs = make(map[string]*list.List)
	ingressTCBpfs = make(map[string]*list.List)
	egressTCBpfs = make(map[string]*list.List)
}

func setupValidBPFList(progName string) *list.List {
	l := list.New()
	bpf := &mocks.BPF{
		MockName: func() string {
			return progName
		},
		MockStop: func(ifaceName, direction string, chain bool) error {
			return nil
		},
		MockUpdateAdminStatus: func(value string) {
			// do nothing
		},
	}
	l.PushBack(bpf)
	l.PushBack(bpf)
	l.PushBack(bpf)
	return l
}

func setupBPFListsWithStopError(progName string) *list.List {
	l := list.New()
	bpf := &mocks.BPF{
		MockName: func() string {
			return progName
		},
		MockStop: func(ifaceName, direction string, chain bool) error {
			return errors.New("stop failure")
		},
		MockUpdateAdminStatus: func(value string) {
			// do nothing
		},
	}
	l.PushBack(bpf)
	return l
}

func setupBPFListsWithPutError(progName string) *list.List {
	l := list.New()
	bpf := &mocks.BPF{
		MockName: func() string {
			return progName
		},
		MockStop: func(ifaceName, direction string, chain bool) error {
			return nil
		},
		MockUpdateAdminStatus: func(value string) {
			// do nothing
		},
		MockProgId: func() int {
			return 0
		},
		MockPutNextProgFDFromID: func(progsID int) error {
			return errors.New("update failure")
		},
		MockGetMapNamePath: func() string {
			return ""
		},
		MockUpdatePrevMapNamePath: func(value string) {
			// do nothing
		},
	}
	l.PushBack(bpf)
	l.PushBack(bpf)
	l.PushBack(bpf)
	return l
}

func setupValidBPF() {
	bpf := BPF{
		Program: &models.BPFProgram{
			ID:                1,
			Name:              "foo",
			Artifact:          "foo.tar.gz",
			CmdStart:          "foo",
			CmdStop:           "",
			Version:           "1.0",
			UserProgramDaemon: true,
			AdminStatus:       "DISABLED",
		},
		Cmd:          nil,
		FilePath:     "",
		RestartCount: 0,
	}
	ifaceName = "dummy"
	seqID = 1
	log.Info().Msg(bpf.Program.Name)
}

func setupBPFProgramData() {
	bpfProgsTmp := &models.BPFPrograms{}
	ifaceName = "dummy"
	seqID = 1

	bpfProg := &models.BPFProgram{
		ID:                1,
		Name:              "foo",
		Artifact:          "foo.tar.gz",
		CmdStart:          "foo",
		CmdStop:           "",
		Version:           "1.0",
		UserProgramDaemon: true,
		AdminStatus:       "ENABLED",
		SeqID:             1,
	}
	bpfProgsTmp.XDPIngress = append(bpfProgsTmp.XDPIngress, bpfProg)

	bpfProgs = bpfProgsTmp
}

func setupBPFProgramVersionChange() {
	bpfProgsTmp := &models.BPFPrograms{}
	ifaceName = "dummy"
	seqID = 1

	bpfProg := &models.BPFProgram{
		ID:                1,
		Name:              "foo",
		Artifact:          "foo.tar.gz",
		CmdStart:          "foo",
		CmdStop:           "",
		Version:           "2.0",
		UserProgramDaemon: true,
		AdminStatus:       "ENABLED",
	}
	bpfProgsTmp.XDPIngress = append(bpfProgsTmp.XDPIngress, bpfProg)
	valVerChange = bpfProgsTmp
}

func setupBPFProgramStatusChange() {

	bpfProgsTmp := &models.BPFPrograms{}
	//cfg := make(map[string][]*models.BPFProgram)
	ifaceName = "dummy"
	seqID = 1

	bpfProg := &models.BPFProgram{
		ID:                1,
		Name:              "foo",
		Artifact:          "foo.tar.gz",
		CmdStart:          "foo",
		CmdStop:           "",
		Version:           "2.0",
		UserProgramDaemon: true,
		AdminStatus:       "DISABLED",
	}
	bpfProgsTmp.XDPIngress = append(bpfProgsTmp.XDPIngress, bpfProg)
	valStatusChange = bpfProgsTmp
}

func TestNewNFConfigs(t *testing.T) {
	type args struct {
		host     string
		hostConf *config.Config
		pMon     *pCheck
		mMon     *kfMetrics
		ctx      context.Context
	}
	setupDBTest()
	hostIfaces, _ := getHostInterfaces()
	tests := []struct {
		name    string
		args    args
		want    *NFConfigs
		wantErr bool
	}{
		{name: "EmptyConfig",
			args: args{
				host:     machineHostname,
				hostConf: nil,
				pMon:     pMon,
				mMon:     mMon},
			want: &NFConfigs{HostName: machineHostname,
				hostInterfaces: hostIfaces,
				IngressXDPBpfs: ingressXDPBpfs,
				IngressTCBpfs:  ingressTCBpfs,
				EgressTCBpfs:   egressTCBpfs,
				HostConfig:     nil,
				processMon:     pMon,
				kfMetricsMon:   mMon,
				mu:             new(sync.Mutex),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNFConfigs(tt.args.ctx, tt.args.host, tt.args.hostConf, tt.args.pMon, tt.args.mMon)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNFConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNFConfigs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNFConfigs_Deploy(t *testing.T) {
	type fields struct {
		hostName       string
		hostInterfaces map[string]bool
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
		metricsMon     *kfMetrics
	}
	type args struct {
		iface    string
		hostName string
		bpfProgs *models.BPFPrograms
	}

	setupDBTest()
	setupValidBPF()
	setupBPFProgramData()
	setupBPFProgramVersionChange()
	setupBPFProgramStatusChange()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "EmptyBPFs",
			fields: fields{
				hostName:       machineHostname,
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     nil,
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "",
				hostName: machineHostname,
				bpfProgs: nil,
			},
			wantErr: true,
		},
		{
			name: "InvalidHostName",
			fields: fields{
				hostName:       machineHostname,
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     nil,
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "dummy",
				hostName: "dummy",
				bpfProgs: bpfProgs,
			},
			wantErr: true,
		},
		{
			name: "ValidHostNameInvalidIfaceName",
			fields: fields{
				hostName:       machineHostname,
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     nil,
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "dummy",
				hostName: machineHostname,
				bpfProgs: &models.BPFPrograms{},
			},
			wantErr: true,
		},
		{
			name: "ValidHostNameValidIfaceName",
			fields: fields{
				hostName:       machineHostname,
				hostInterfaces: map[string]bool{"fakeif0": true},
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     nil,
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "fakeif0",
				hostName: machineHostname,
				bpfProgs: &models.BPFPrograms{},
			},
			wantErr: false,
		},
		{
			name: "TestEBPFRepoDownload",
			fields: fields{
				hostName:       machineHostname,
				hostInterfaces: map[string]bool{"fakeif0": true},
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     &config.Config{BPFDir: "/tmp", EBPFRepoURL: "http://www.example.com"},
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "fakeif0",
				hostName: machineHostname,
				bpfProgs: bpfProgs,
			},
			wantErr: false,
		},
		{
			name: "NewBPFWithVersionChange",
			fields: fields{
				hostName:       machineHostname,
				hostInterfaces: map[string]bool{"fakeif0": true},
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     &config.Config{BPFDir: "/tmp", EBPFRepoURL: "http://www.example.com"},
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "fakeif0",
				hostName: machineHostname,
				bpfProgs: valVerChange,
			},
			wantErr: false,
		},
		{
			name: "NewBPFWithStatusChange",
			fields: fields{
				hostName:       machineHostname,
				hostInterfaces: map[string]bool{"fakeif0": true},
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig:     &config.Config{BPFDir: "/tmp", EBPFRepoURL: "http://www.example.com"},
				processMon:     pMon,
				metricsMon:     mMon,
			},
			args: args{
				iface:    "fakeif0",
				hostName: machineHostname,
				bpfProgs: valStatusChange,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName: tt.fields.hostName,
				//				configs:    tt.fields.configs,
				hostInterfaces: tt.fields.hostInterfaces,
				IngressXDPBpfs: tt.fields.ingressXDPBpfs,
				IngressTCBpfs:  tt.fields.ingressTCBpfs,
				EgressTCBpfs:   tt.fields.egressTCBpfs,
				HostConfig:     tt.fields.hostConfig,
				processMon:     tt.fields.processMon,
				mu:             new(sync.Mutex),
			}
			if err := cfg.Deploy(tt.args.iface, tt.args.hostName, tt.args.bpfProgs); (err != nil) != tt.wantErr {
				t.Errorf("NFConfigs.Deploy() error = %#v, wantErr %#v", err, tt.wantErr)
			}
		})
	}
}

func TestNFConfigs_Close(t *testing.T) {
	type fields struct {
		hostName       string
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "EmptyMap",
			fields: fields{
				hostName:       machineHostname,
				ingressXDPBpfs: make(map[string]*list.List),
				ingressTCBpfs:  make(map[string]*list.List),
				egressTCBpfs:   make(map[string]*list.List),
				hostConfig: &config.Config{
					BpfMapDefaultPath: "/sys/fs/bpf",
				},
				processMon: pMon,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:       tt.fields.hostName,
				IngressXDPBpfs: tt.fields.ingressXDPBpfs,
				IngressTCBpfs:  tt.fields.ingressTCBpfs,
				EgressTCBpfs:   tt.fields.egressTCBpfs,
				HostConfig:     tt.fields.hostConfig,
				processMon:     tt.fields.processMon,
			}
			ctx, cancelfunc := context.WithCancel(context.Background())
			defer cancelfunc()
			if err := cfg.Close(ctx); (err != nil) != tt.wantErr {
				t.Errorf("NFConfigs.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getHostInterfaces(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "GoodInput",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getHostInterfaces()
			if (err != nil) != tt.wantErr {
				t.Errorf("getHostInterfaces() error : %v", err)
			}
		})
	}
}
func Test_BinarySearch(t *testing.T) {
	tests := []struct {
		name   string
		vals   []string
		target string
		result bool
	}{
		{
			name:   "FoundTheTarget",
			vals:   []string{"connection-limit", "ipfix-flow-exporter", "ratelimiting"},
			target: "ratelimiting",
			result: true,
		},
		{
			name:   "DidNotFindTheTarget",
			vals:   []string{"connection-limit", "ipfix-flow-exporter", "ratelimiting"},
			target: "zsdf",
			result: false,
		},
	}

	for _, tt := range tests {
		if BinarySearch(tt.vals, tt.target) != tt.result {
			t.Errorf("BinarySearch is not producing expected output")
		}
	}
}

func TestAddeBPDProgramsByHook(t *testing.T) {
	mocked := errors.New("mocked")
	type fields struct {
		hostname            string
		hostInterfaces      map[string]bool
		ingressXDPBpfs      map[string]*list.List
		ingressTCBpfs       map[string]*list.List
		egressTCBpfs        map[string]*list.List
		hostConfig          *config.Config
		testLoadRootProgram func(ifaceName, direction, progType string, conf *config.Config) (models.BPF, error)
	}
	type args struct {
		iface     string
		direction string
		progMap   map[string]*list.List
		bpfProgs  []*models.BPFProgram
	}
	tests := []struct {
		name    string
		field   fields
		arg     args
		wanterr bool
	}{
		{
			name:  "NonNilProgMap",
			field: fields{},
			arg: args{
				iface:   "fakeif0",
				progMap: map[string]*list.List{"fakeif0": setupValidBPFList("mocked")},
				bpfProgs: []*models.BPFProgram{
					&models.BPFProgram{},
				},
			},
			wanterr: true,
		},
		{
			name:  "AdminStatusDisabled",
			field: fields{},
			arg: args{
				iface:   "fakeif0",
				progMap: map[string]*list.List{"fakeif0": nil},
				bpfProgs: []*models.BPFProgram{
					&models.BPFProgram{
						AdminStatus: models.Disabled,
					},
				},
			},
			wanterr: false,
		},
		{
			name: "UnknownDirection",
			arg: args{
				iface:   "fakeif0",
				progMap: map[string]*list.List{"fakeif0": nil},
				bpfProgs: []*models.BPFProgram{
					&models.BPFProgram{
						AdminStatus: models.Enabled,
					},
				},
			},
			wanterr: true,
		},
		{
			name: "BadInput",
			field: fields{
				hostname:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
				hostConfig: &config.Config{
					BpfChainingEnabled: false,
				},
				testLoadRootProgram: func(ifaceName, direction, progType string, conf *config.Config) (models.BPF, error) {
					return &mocks.BPF{}, mocked
				},
			},
			arg: args{
				iface:     "fakeif0",
				direction: models.XDPIngressType,
				progMap: map[string]*list.List{
					"fakeif0": nil,
				},
				bpfProgs: []*models.BPFProgram{
					&models.BPFProgram{
						Name:              "dummy_name",
						SeqID:             1,
						Artifact:          "dummy_artifact.tar.gz",
						MapName:           "xdp_rl_ingress_next_prog",
						CmdStart:          "dummy_command",
						Version:           "latest",
						UserProgramDaemon: true,
						AdminStatus:       models.Enabled,
						ProgType:          "xdp",
						CfgVersion:        1,
					},
				},
			},
			wanterr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:            tt.field.hostname,
				hostInterfaces:      tt.field.hostInterfaces,
				IngressXDPBpfs:      tt.field.ingressXDPBpfs,
				IngressTCBpfs:       tt.field.ingressTCBpfs,
				EgressTCBpfs:        tt.field.egressTCBpfs,
				HostConfig:          tt.field.hostConfig,
				TestLoadRootProgram: tt.field.testLoadRootProgram,
			}
			err := cfg.AddeBPFProgramsByHook(tt.arg.iface, tt.arg.direction, tt.arg.progMap, tt.arg.bpfProgs)
			if (err != nil) != tt.wanterr {
				t.Errorf("AddeBPFProgramsByHook: %v", err)
			}
		})
	}
}

func Test_AddProgramsOnInterface(t *testing.T) {
	type fields struct {
		hostName       string
		hostInterfaces map[string]bool
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
		mu             *sync.Mutex
	}
	type args struct {
		iface    string
		hostName string
		bpfProgs *models.BPFPrograms
	}
	tests := []struct {
		name    string
		field   fields
		arg     args
		wanterr bool
	}{
		{
			name:    "UnknownHostName",
			field:   fields{},
			arg:     args{},
			wanterr: true,
		},
		{
			name: "NilInterface",
			field: fields{
				hostName: "l3af-local-test",
			},
			arg: args{
				hostName: "fakeif0",
			},
			wanterr: true,
		},
		{
			name: "UnknownInterface",
			field: fields{
				hostName: "l3af-local-test",
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "dummyinterface",
				bpfProgs: &models.BPFPrograms{
					XDPIngress: []*models.BPFProgram{
						&models.BPFProgram{
							Name:              "dummy_name",
							SeqID:             1,
							Artifact:          "dummy_artifact.tar.gz",
							MapName:           "xdp_rl_ingress_next_prog",
							CmdStart:          "dummy_command",
							Version:           "latest",
							UserProgramDaemon: true,
							AdminStatus:       "enabled",
							ProgType:          "xdp",
							CfgVersion:        1,
						},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "GoodInput",
			field: fields{
				hostName:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				mu:             new(sync.Mutex),
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "fakeif0",
				bpfProgs: &models.BPFPrograms{
					XDPIngress: []*models.BPFProgram{},
					TCEgress:   []*models.BPFProgram{},
					TCIngress:  []*models.BPFProgram{},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:       tt.field.hostName,
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				HostConfig:     tt.field.hostConfig,
				processMon:     tt.field.processMon,
				hostInterfaces: tt.field.hostInterfaces,
				mu:             tt.field.mu,
			}
			err := cfg.AddProgramsOnInterface(tt.arg.iface, tt.arg.hostName, tt.arg.bpfProgs)
			if (err != nil) != tt.wanterr {
				t.Errorf("AddProgramsOnInterface: %v", err)
			}
		})
	}
}

func TestAddeBPFPrograms(t *testing.T) {
	type fields struct {
		hostName       string
		hostInterfaces map[string]bool
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
		mu             *sync.Mutex
		ifaces         map[string]string
	}
	tests := []struct {
		name    string
		field   fields
		arg     []models.L3afBPFPrograms
		wanterr bool
	}{
		{
			name: "UnknownHostName",
			field: fields{
				hostName: "l3af-prod",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFPrograms{
				{
					HostName: "l3af-test",
					Iface:    "fakeif0",
					BpfPrograms: &models.BPFPrograms{
						XDPIngress: []*models.BPFProgram{},
						TCIngress:  []*models.BPFProgram{},
						TCEgress:   []*models.BPFProgram{},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "NilInterface",
			field: fields{
				hostName: "l3af-local-test",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFPrograms{
				{
					HostName: "l3af-local-test",
					BpfPrograms: &models.BPFPrograms{
						XDPIngress: []*models.BPFProgram{},
						TCIngress:  []*models.BPFProgram{},
						TCEgress:   []*models.BPFProgram{},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "UnknownInterface",
			field: fields{
				hostName: "l3af-local-test",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFPrograms{
				{
					HostName: "l3af-local-test",
					Iface:    "dummyinterface",
					BpfPrograms: &models.BPFPrograms{
						XDPIngress: []*models.BPFProgram{
							&models.BPFProgram{
								Name:              "dummy_name",
								SeqID:             1,
								Artifact:          "dummy_artifact_name",
								MapName:           "xdp_rl_ingress_next_prog",
								CmdStart:          "dummy_command",
								Version:           "latest",
								UserProgramDaemon: true,
								AdminStatus:       "enabled",
								ProgType:          "xdp",
								CfgVersion:        1,
							},
						},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "GoodInput",
			field: fields{
				hostName:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				// fakeif0 is a fake interface
				mu:             new(sync.Mutex),
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
				ifaces:         map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFPrograms{
				{
					HostName: "l3af-local-test",
					Iface:    "fakeif0",
					BpfPrograms: &models.BPFPrograms{
						XDPIngress: []*models.BPFProgram{},
						TCIngress:  []*models.BPFProgram{},
						TCEgress:   []*models.BPFProgram{},
					},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:       tt.field.hostName,
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				HostConfig:     tt.field.hostConfig,
				processMon:     tt.field.processMon,
				hostInterfaces: tt.field.hostInterfaces,
				mu:             tt.field.mu,
			}
			err := cfg.AddeBPFPrograms(tt.arg)
			if (err != nil) != tt.wanterr {
				t.Errorf("AddeBPFPrograms failed: %v", err)
			}
		})
	}
}

func TestDeleteProgramsOnInterfaceHelper(t *testing.T) {
	mockedErr := errors.New("mocked")
	mockedList := setupBPFListsWithPutError("mocked")
	mockedValidList := setupValidBPFList("mocked")
	type fields struct {
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
	}
	type args struct {
		e         *list.Element
		iface     string
		direction string
		bpfList   *list.List
	}
	tests := []struct {
		name    string
		field   fields
		arg     args
		wanterr bool
	}{
		{
			name: "NilProgram",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				e: nil,
			},
			wanterr: false,
		},
		{
			name: "StopError",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				e: &list.Element{
					Value: &mocks.BPF{
						MockName: func() string {
							return "mocked"
						},
						MockStop: func(ifaceName, direction string, chain bool) error {
							return mockedErr
						},
						MockUpdateAdminStatus: func(value string) {
							// do nothing
						},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "BpfChainingDisabled",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: false,
				},
			},
			arg:     args{},
			wanterr: false,
		},
		{
			name: "NextNotNil",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				e:       mockedList.Front().Next(),
				bpfList: mockedList,
			},
			wanterr: true,
		},
		{
			name: "UnknownDirection",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				e:         mockedValidList.Back(),
				bpfList:   mockedValidList,
				direction: "random",
			},
			wanterr: false,
		},
		{
			name: "GoodInput",
			field: fields{
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				e:         mockedValidList.Front().Next(),
				iface:     "fakeif0",
				direction: models.XDPIngressType,
				bpfList:   mockedValidList,
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				HostConfig:     tt.field.hostConfig,
			}
			err := cfg.DeleteProgramsOnInterfaceHelper(tt.arg.e, tt.arg.iface, tt.arg.direction, tt.arg.bpfList)
			if (err != nil) != tt.wanterr {
				t.Errorf("deleteProgramsOnInterfaceHelper failed: %v", err)
			}
		})
	}

}

func TestDeleteProgramsByHook(t *testing.T) {
	type args struct {
		iface     string
		direction string
		progMap   map[string]*list.List
		bpfProgs  []string
	}
	tests := []struct {
		name    string
		arg     args
		wanterr bool
	}{
		{
			name: "NilProgMap",
			arg: args{
				progMap: nil,
			},
			wanterr: false,
		},
		{
			name: "UnkownInterface",
			arg: args{
				iface: "fakeif0",
			},
			wanterr: false,
		},
		{
			name: "NilBPFList",
			arg: args{
				iface:   "fakeif0",
				progMap: map[string]*list.List{"fakeif0": nil},
			},
			wanterr: false,
		},
		{
			name: "UnknownBPFProgs",
			arg: args{
				iface: "fakeif0",
				progMap: map[string]*list.List{
					"fakeif0": setupBPFListsWithStopError("ratelimiting"),
				},
				bpfProgs: []string{"random"},
			},
			wanterr: false,
		},
		{
			name: "BadInput",
			arg: args{
				iface: "fakeif0",
				progMap: map[string]*list.List{
					"fakeif0": setupBPFListsWithStopError("ratelimiting"),
				},
				bpfProgs: []string{"ratelimiting"},
			},
			wanterr: true,
		},
		{
			name: "GoodInput",
			arg: args{
				iface: "fakeif0",
				progMap: map[string]*list.List{
					"fakeif0": list.New(),
				},
				bpfProgs: []string{"ratelimiting"},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostConfig: &config.Config{},
			}
			err := cfg.DeleteProgramsByHook(tt.arg.iface, tt.arg.direction, tt.arg.progMap, tt.arg.bpfProgs)
			if (err != nil) != tt.wanterr {
				t.Errorf("deleteProgramsByHook failed: %v", err)
			}
		})
	}
}

func TestDeleteProgramsOnInterface(t *testing.T) {
	type fields struct {
		hostName       string
		hostInterfaces map[string]bool
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
		mu             *sync.Mutex
	}
	type args struct {
		iface    string
		hostName string
		bpfProgs *models.BPFProgramNames
	}
	tests := []struct {
		name    string
		field   fields
		arg     args
		wanterr bool
	}{
		{
			name:  "UnknownHostName",
			field: fields{},
			arg: args{
				hostName: "l3af-local-test",
			},
			wanterr: true,
		},
		{
			name: "NilInterface",
			field: fields{
				hostName: "l3af-local-test",
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "",
			},
			wanterr: true,
		},
		{
			name: "NilBPFProgs",
			field: fields{
				hostName: "l3af-local-test",
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "dummyinterface",
				bpfProgs: nil,
			},
			wanterr: true,
		},
		{
			name: "UnknownInterface",
			field: fields{
				hostName: "l3af-local-test",
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "dummyinterface",
				bpfProgs: &models.BPFProgramNames{
					XDPIngress: []string{},
					TCIngress:  []string{},
					TCEgress:   []string{},
				},
			},
			wanterr: true,
		},
		{
			name: "BadInput",
			field: fields{
				hostName:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				mu:             new(sync.Mutex),
				ingressXDPBpfs: map[string]*list.List{
					"fakeif0": setupBPFListsWithStopError("ratelimiting"),
				},
				ingressTCBpfs: map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				hostConfig: &config.Config{
					BPFLogDir:  "",
					DataCenter: "localdc",
				},
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "fakeif0",
				bpfProgs: &models.BPFProgramNames{
					XDPIngress: []string{"ratelimiting"},
					TCIngress:  []string{},
					TCEgress:   []string{},
				},
			},
			wanterr: true,
		},
		{
			name: "GoodInput",
			field: fields{
				hostName:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				mu:             new(sync.Mutex),
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
			},
			arg: args{
				hostName: "l3af-local-test",
				iface:    "fakeif0",
				bpfProgs: &models.BPFProgramNames{
					XDPIngress: []string{},
					TCIngress:  []string{},
					TCEgress:   []string{},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:       tt.field.hostName,
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				HostConfig:     tt.field.hostConfig,
				processMon:     tt.field.processMon,
				hostInterfaces: tt.field.hostInterfaces,
				mu:             tt.field.mu,
			}
			err := cfg.DeleteProgramsOnInterface(tt.arg.iface, tt.arg.hostName, tt.arg.bpfProgs)
			if (err != nil) != tt.wanterr {
				t.Errorf("DeleteProgramsOnInterface failed: %v", err)
			}
		})
	}
}

func TestDeleteEbpfPrograms(t *testing.T) {
	type fields struct {
		hostName       string
		hostInterfaces map[string]bool
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
		processMon     *pCheck
		mu             *sync.Mutex
		ifaces         map[string]string
	}
	tests := []struct {
		name    string
		field   fields
		arg     []models.L3afBPFProgramNames
		wanterr bool
	}{
		{
			name: "UnknowhostName",
			field: fields{
				hostName: "l3af-prod",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFProgramNames{
				{
					HostName: "l3af-local-test",
					Iface:    "fakeif0",
					BpfProgramNames: &models.BPFProgramNames{
						XDPIngress: []string{},
						TCIngress:  []string{},
						TCEgress:   []string{},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "NilInterface",
			field: fields{
				hostName: "l3af-local-test",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFProgramNames{
				{
					HostName: "l3af-local-test",
					Iface:    "fakeif0",
					BpfProgramNames: &models.BPFProgramNames{
						XDPIngress: []string{},
						TCIngress:  []string{},
						TCEgress:   []string{},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "UnknownInterface",
			field: fields{
				hostName: "l3af-local-test",
				ifaces:   map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFProgramNames{
				{
					HostName: "l3af-local-test",
					Iface:    "fakeif0",
					BpfProgramNames: &models.BPFProgramNames{
						XDPIngress: []string{},
						TCIngress:  []string{},
						TCEgress:   []string{},
					},
				},
			},
			wanterr: true,
		},
		{
			name: "GoodInput",
			field: fields{
				hostName:       "l3af-local-test",
				hostInterfaces: map[string]bool{"fakeif0": true},
				mu:             new(sync.Mutex),
				ingressXDPBpfs: map[string]*list.List{"fakeif0": nil},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": nil},
				egressTCBpfs:   map[string]*list.List{"fakeif0": nil},
				ifaces:         map[string]string{},
				hostConfig: &config.Config{
					L3afConfigStoreFileName: filepath.FromSlash("../testdata/Test_l3af-config.json"),
				},
			},
			arg: []models.L3afBPFProgramNames{
				{
					HostName: "l3af-local-test",
					Iface:    "fakeif0",
					BpfProgramNames: &models.BPFProgramNames{
						XDPIngress: []string{},
						TCIngress:  []string{},
						TCEgress:   []string{},
					},
				},
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostName:       tt.field.hostName,
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				HostConfig:     tt.field.hostConfig,
				processMon:     tt.field.processMon,
				hostInterfaces: tt.field.hostInterfaces,
				mu:             tt.field.mu,
			}
			err := cfg.DeleteEbpfPrograms(tt.arg)
			if (err != nil) != tt.wanterr {
				t.Errorf("DeleteEbpfPrograms failed: %v", err)
			}
		})
	}
}

func TestAddAndStartBPF(t *testing.T) {
	type field struct {
		ctx        context.Context
		hostConfig *config.Config
	}
	type arg struct {
		bpfProg   *models.BPFProgram
		direction string
		iface     string
	}
	tests := []struct {
		name    string
		fields  field
		args    arg
		wanterr bool
	}{
		{
			name:   "NilProgram",
			fields: field{},
			args: arg{
				bpfProg:   nil,
				direction: "fakedirection",
				iface:     "fakeif0",
			},
			wanterr: true,
		},
		{
			name:   "AdminStatusDisabled",
			fields: field{},
			args: arg{
				bpfProg: &models.BPFProgram{
					Name:        "dummy",
					AdminStatus: "disabled",
				},
				direction: "fakedirection",
				iface:     "fakeif0",
			},
			wanterr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				ctx:        tt.fields.ctx,
				HostConfig: tt.fields.hostConfig,
			}
			e := cfg.AddAndStartBPF(tt.args.bpfProg, tt.args.iface, tt.args.direction)
			if (e != nil) != tt.wanterr {
				t.Errorf("AddAndStartBPF failed : %v", e)
			}
		})
	}
}

func TestAddProgramWithoutChaining(t *testing.T) {
	progList := list.New()
	progList.PushBack(&BPF{
		Program: &models.BPFProgram{
			Name: "dummyProgram",
		},
	})
	type fields struct {
		ingressXDPBpfs map[string]*list.List
		ingressTCBpfs  map[string]*list.List
		egressTCBpfs   map[string]*list.List
		hostConfig     *config.Config
	}
	type args struct {
		iface    string
		hostName string
		bpfProgs *models.BPFPrograms
	}
	tests := []struct {
		name    string
		field   fields
		arg     args
		wanterr bool
	}{
		{
			name: "chainingEnabled",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: true,
				},
			},
			arg: args{
				iface:    "fakeif0",
				hostName: "fakehost",
			},
			wanterr: false,
		},
		{
			name: "badInput",
			field: fields{
				hostConfig: &config.Config{
					BpfChainingEnabled: false,
				},
				ingressXDPBpfs: map[string]*list.List{"fakeif0": progList},
				egressTCBpfs:   map[string]*list.List{"fakeif0": progList},
				ingressTCBpfs:  map[string]*list.List{"fakeif0": progList},
			},
			arg: args{
				iface: "fakeif0",
				bpfProgs: &models.BPFPrograms{
					XDPIngress: []*models.BPFProgram{
						{
							Name:        "dummyProgram",
							AdminStatus: models.Enabled,
						},
					},
					TCIngress: []*models.BPFProgram{
						{
							Name:        "dummyProgram",
							AdminStatus: models.Enabled,
						},
					},
					TCEgress: []*models.BPFProgram{
						{
							Name:        "dummyProgram",
							AdminStatus: models.Enabled,
						},
					},
				},
			},
			wanterr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &NFConfigs{
				HostConfig:     tt.field.hostConfig,
				IngressXDPBpfs: tt.field.ingressXDPBpfs,
				EgressTCBpfs:   tt.field.egressTCBpfs,
				IngressTCBpfs:  tt.field.ingressTCBpfs,
			}
			e := cfg.AddProgramWithoutChaining(tt.arg.iface, tt.arg.bpfProgs)
			if (e != nil) != tt.wanterr {
				t.Errorf(" AddProgramWithoutChaining failed : %v", e)
			}
		})
	}
}
