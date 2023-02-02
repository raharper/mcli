package api

import (
	"fmt"
	"mcli-v2/pkg/qcli"
	"os"
	"path"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func GetKvmPath() (string, error) {
	// perfer qemu-kvm, qemu-system-x86_64, kvm
	emulators := []string{"qemu-kvm", "qemu-system-x86_64", "kvm"}
	paths := []string{"/usr/libexec", "/usr/bin"}

	for _, emulator := range emulators {
		for _, prefix := range paths {
			kvmPath := path.Join(prefix, emulator)
			if _, err := os.Stat(kvmPath); err == nil {
				return kvmPath, nil
			}
		}
	}
	return "", fmt.Errorf("Failed to find QEMU/KVM binary [%s] in paths [%s]\n", emulators, paths)
}

func NewDefaultConfig(name string, numCpus, numMemMB uint32) (*qcli.Config, error) {
	smp := qcli.SMP{CPUs: numCpus}
	if numCpus < 1 {
		smp.CPUs = 4
	}

	mem := qcli.Memory{
		Size: fmt.Sprintf("%dm", numMemMB),
	}
	if numMemMB < 1 {
		mem.Size = "4096m"
	}

	path, err := GetKvmPath()
	if err != nil {
		return &qcli.Config{}, fmt.Errorf("Failed creating new default config: %s", err)
	}

	c := &qcli.Config{
		Name: name,
		Path: path,
		Machine: qcli.Machine{
			Type:         qcli.MachineTypePC35,
			Acceleration: qcli.MachineAccelerationKVM,
			SMM:          "on",
		},
		CPUModel:      "qemu64",
		CPUModelFlags: []string{"+x2apic"},
		SMP:           smp,
		Memory:        mem,
		RngDevices: []qcli.RngDevice{
			qcli.RngDevice{
				Driver:    qcli.VirtioRng,
				ID:        "rng0",
				Bus:       "pcie.0",
				Transport: qcli.TransportPCI,
				Filename:  qcli.RngDevUrandom,
			},
		},
		GlobalParams: []string{
			"ICH9-LPC.disable_s3=1",
			"driver=cfi.pflash01,property=secure,value=on",
		},
	}

	return c, nil
}

// FIXME: what to do with remote client/server ? push to zot and use zot URLs?
// ImportDiskImage will copy/create a source image to server image
func (qd *QemuDisk) ImportDiskImage(imageDir string) error {

	// What to do about sparse? use reflink and sparse=auto for now.
	srcFilePath := qd.File
	destFilePath := filepath.Join(imageDir, filepath.Base(srcFilePath))
	qd.File = destFilePath

	if qd.Size > 0 {
		return qd.Create()
	} else {
		log.Infof("Importing VM disk '%s' -> '%s'", srcFilePath, destFilePath)
		err := CopyFileRefSparse(srcFilePath, destFilePath)
		if err != nil {
			return fmt.Errorf("Error copying VM disk '%s' -> '%s': %s", srcFilePath, destFilePath, err)
		}
	}

	return nil
}

func (qd *QemuDisk) QBlockDevice() (qcli.BlockDevice, error) {
	blk := qcli.BlockDevice{
		// Driver
		ID:        fmt.Sprintf("drive%d", getNextQemuIndex("drive")),
		File:      qd.File,
		Interface: qcli.NoInterface,
		AIO:       qcli.Threads,
		BlockSize: qd.BlockSize,
		BusAddr:   qd.BusAddr,
		BootIndex: qd.BootIndex,
		ReadOnly:  qd.ReadOnly,
	}

	if qd.Format != "" {
		switch qd.Format {
		case "raw":
			blk.Format = qcli.RAW
		case "qcow2":
			blk.Format = qcli.QCOW2
		}
	} else {
		blk.Format = qcli.QCOW2
	}

	if qd.Attach == "" {
		qd.Attach = "virtio"
	}

	switch qd.Attach {
	case "scsi":
		blk.Driver = qcli.SCSIHD
	case "nvme":
		blk.Driver = qcli.NVME
	case "virtio":
		blk.Driver = qcli.VirtioBlock
	case "ide":
		if qd.Type == "cdrom" {
			blk.Driver = qcli.IDECDROM
			blk.Media = "cdrom"
		} else {
			blk.Driver = qcli.IDEHardDisk
		}
	case "usb":
		blk.Driver = qcli.USBStorage
	default:
		return blk, fmt.Errorf("Unknown Disk Attach type: %s", qd.Attach)
	}

	return blk, nil
}

func (nd NicDef) QNetDevice() (qcli.NetDevice, error) {
	//FIXME: how do we do bridge or socket/mcast types?
	ndev := qcli.NetDevice{
		Type:       qcli.USER,
		ID:         nd.ID,
		Addr:       nd.BusAddr,
		MACAddress: nd.Mac,
		User: qcli.NetDeviceUser{
			IPV4: true,
		},
		BootIndex: nd.BootIndex,
		Driver:    qcli.DeviceDriver(nd.Device),
	}
	return ndev, nil
}

func GenerateQConfig(runDir string, v VMDef) (*qcli.Config, error) {
	c, err := NewDefaultConfig(v.Name, v.Cpus, v.Memory)
	if err != nil {
		return c, err
	}

	if !PathExists(runDir) {
		err := EnsureDir(runDir)
		if err != nil {
			return c, fmt.Errorf("Error creating VM run dir '%s': %s", runDir, err)
		}
	}

	if v.Cdrom != "" {
		qd := QemuDisk{
			File:   v.Cdrom,
			Format: "raw",
			Attach: "ide",
			Type:   "cdrom",
		}
		v.Disks = append(v.Disks, qd)
	}

	v.AdjustBootIndicies()

	for i := range v.Disks {
		var disk *QemuDisk
		disk = &v.Disks[i]

		if err := disk.Sanitize(runDir); err != nil {
			return c, err
		}

		// import/create files into stateDir/images/basename(File)
		if err := disk.ImportDiskImage(runDir); err != nil {
			return c, err
		}

		qblk, err := disk.QBlockDevice()
		if err != nil {
			return c, err
		}
		c.BlkDevices = append(c.BlkDevices, qblk)
	}

	for _, nic := range v.Nics {
		qnet, err := nic.QNetDevice()
		if err != nil {
			return c, err
		}
		c.NetDevices = append(c.NetDevices, qnet)
	}

	return c, nil
}

type QMPMachineLogger struct{}

func (l QMPMachineLogger) V(level int32) bool {
	return true
}

func (l QMPMachineLogger) Infof(format string, v ...interface{}) {
	log.Infof(format, v...)
}

func (l QMPMachineLogger) Warningf(format string, v ...interface{}) {
	log.Warnf(format, v...)
}

func (l QMPMachineLogger) Errorf(format string, v ...interface{}) {
	log.Errorf(format, v...)
}
