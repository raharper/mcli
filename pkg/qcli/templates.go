package qcli

import "fmt"

func NewDefaultConfig(numCpus, numMemMB uint32) (*Config, error) {
	smp := SMP{CPUs: numCpus}
	if numCpus < 1 {
		smp.CPUs = 4
	}

	mem := Memory{
		Size: fmt.Sprintf("%dm", numMemMB),
	}
	if numMemMB < 1 {
		mem.Size = "4096m"
	}

	c := &Config{
		Machine: Machine{
			Type:         MachineTypePC35,
			Acceleration: MachineAccelerationKVM,
			SMM:          "on",
		},
		CPUModel:      "qemu64",
		CPUModelFlags: []string{"+x2apic"},
		SMP:           smp,
		Memory:        mem,
		RngDevices: []RngDevice{
			RngDevice{
				Driver:    VirtioRng,
				ID:        "rng0",
				Bus:       "pcie.0",
				Addr:      "3",
				Transport: TransportPCI,
				Filename:  RngDevUrandom,
			},
		},
		GlobalParams: []string{
			"ICH9-LPC.disable_s3=1",
			"driver=cfi.pflash01,property=secure,value=on",
		},
	}

	return c, nil
}
