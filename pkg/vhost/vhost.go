package vhost

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PVcreate - Use PVcreate to mark disk as LVM physical volumes
func PVcreate(diskPath string) error {
	//pvcreate /dev/sdb1
	output, err := exec.Command("pvcreate", diskPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pvcreate: err:[%w] output:[%s]", err, output)
	}

	return nil
}

// VGcreate - Make LVM physical volumes into volume groups
func VGcreate(diskPath, vgName string) error {
	// vgcreate testvg /dev/sdb1
	output, err := exec.Command("vgcreate", vgName, diskPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to vgcreate: err:[%w] output:[%s]", err, output)
	}

	return nil
}

// LVcreate - Create a logical volume on the volume group
func LVcreate(lvName, vgName string, sizeDisk int) error {
	// lvcreate -L 50G --name testlv testvg
	output, err := exec.Command("lvcreate", "-L",
								fmt.Sprintf("%dG", sizeDisk), "--name",
								lvName, vgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to LVcreate: err:[%w] output:[%s]", err, output)
	}

	return nil
}

// PVremove - Use LVremove to remove the disk from as LVM physical volumes
func PVremove(targetDisk string) error {
	//pvremove /dev/sdb1
	output, err := exec.Command("pvremove", "-y", targetDisk).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pvremove: err:[%w] output:[%s]", err, output)
	}
	return nil
}

// VGremove - Remove volume groups
func VGremove(vgName string) error {
	//vgremove testvg
	output, err := exec.Command("vgremove", "-y", vgName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to vgremove: err:[%w] output:[%s]", err, output)
	}
	return nil
}

// LVremove - remove a logical volume
func LVremove(lvName, vgName string) error {
	//lvremove /dev/testvg/testlv
	lvpath := filepath.Join("/dev/", vgName, lvName)
	output, err := exec.Command("lvremove", "-y", lvpath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to lvremove: err:[%w] output:[%s]", err, output)
	}
	return nil
}

// DestroyLvm - Remove volume groups and marker LVM on physical volumes
func DestroyLvm(targetDisk, vgName string) error {
	if err := VGremove(vgName); err != nil {
		return fmt.Errorf("VGremove failed err:[%w]", err)
	}

	if err := PVremove(targetDisk); err != nil {
		return fmt.Errorf("PVremove failed err:[%w]", err)
	}
	return nil
}

func CheckConfigFS() error {
	if _, err := os.Stat(tgtPath); err != nil {
		return fmt.Errorf("target access error (%s): %v", tgtPath, err)
	}
	if _, err := os.Stat(corePath); err != nil {
		return fmt.Errorf("target core access error (%s): %s", corePath, err)
	}
	if _, err := os.Stat(vhostPath); err != nil {
		return fmt.Errorf("vHOST access error (%s): %s", vhostPath, err)
	}
	return nil
}

func CheckLvmOnSystem() error {
	output, err := exec.Command("lvm", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to collect tools data (lvm)! output:[%s] err:[%w]", output, err)
	}
	if err := CheckConfigFS(); err != nil {
		return fmt.Errorf("failed to checked ConfigFS! output:[%s] err:[%w]", output, err)
	}
	return nil
}

func CheckZfsOnSystem() error {
	output, err := exec.Command("zfs", "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to collect tools data (zfs)! output:[%s] err:[%w]", output, err)
	}
	if err := CheckConfigFS(); err != nil {
		return fmt.Errorf("failed to checked ConfigFS! output:[%s] err:[%w]", output, err)
	}
	return nil
}

// Write a function to handle partitioning disks

// CreateZpool for update option
func CreateZpool(zpoolName, targetDisk string) (error) {
	// Workaround if something went wrong with specifying parameters
	output, err := exec.Command("zpool", "create", "-fd", zpoolName, targetDisk).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create zpool: err:[%w] output:[%s]", err, output)
	}
	return nil
}

func DestroyZpool(zpoolName string) error {
	// Need handle to pool at first place
	output, err := exec.Command("zpool", "destroy", zpoolName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to destroy zvol: log:%s err:%w", output, err)
	}
	return nil
}

func CreateZvol(zpoolName, zvolName, bs, zip, prcache, logbias,
				redundant_metadata string, sizeDisk int) error {
	//zfs create -V 1G tank/disk1
	output, err := exec.Command("zfs", "create",
								"-V", fmt.Sprintf("%dG", sizeDisk),
								"-o", fmt.Sprintf("volblocksize=%s", bs),
								"-o", fmt.Sprintf("compression=%s", zip),
								"-o", fmt.Sprintf("primarycache=%s", prcache),
								"-o", fmt.Sprintf("logbias=%s", logbias),
								"-o", fmt.Sprintf("redundant_metadata=%s", redundant_metadata),
	 							fmt.Sprintf("%s/%s", zpoolName, zvolName)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create zvol: log:%s err:%w", output, err)
	}
	return nil
}

func DestroyZvol(zpoolName, zvolName string) error {
	output, err := exec.Command("zfs", "destroy",
					fmt.Sprintf("%s/%s", zpoolName, zvolName)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to destroy zvol: log:%s err:%w", output, err)
	}
	return nil
}

func waitForFile(fileName string) error {
	maxDelay := time.Second * 5
	delay := time.Millisecond * 500
	var waited time.Duration
	for {
		if delay != 0 {
			time.Sleep(delay)
			waited += delay
		}
		if _, err := os.Stat(fileName); err == nil {
			return nil
		} else {
			if waited > maxDelay {
				return fmt.Errorf("file not found: error %v", err)
			}
			delay = 2 * delay
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
}

const (
	vhostPath  = "/sys/kernel/config/target/vhost/"
	corePath   = "/sys/kernel/config/target/core/"
	tgtPath    = "/sys/kernel/config/target"
	iBlockPath = tgtPath + "/core/iblock_0"
	naaPrefix  = "5001405" // from rtslib-fb
)

// VHostCreateIBlock - Create vHost fabric
func VHostCreateIBlock(tgtName, wwn string) error {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	if _, err := os.Stat(targetRoot); err != nil {
		return fmt.Errorf("tgt access error (%s): %s", targetRoot, err)
	}
	vhostRoot := filepath.Join(tgtPath, "vhost", wwn, "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	err := os.MkdirAll(vhostLun, os.ModeDir)
	if err != nil {
		return fmt.Errorf("cannot create vhost: %v", err)
	}
	controlCommand := "scsi_host_id=1,scsi_channel_id=0,scsi_target_id=0,scsi_lun_id=0"
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "control"), []byte(controlCommand), 0660); err != nil {
		return fmt.Errorf("error set control: %v", err)
	}
	if err := waitForFile(filepath.Join(vhostRoot, "nexus")); err != nil {
		return fmt.Errorf("error waitForFile: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(vhostRoot, "nexus"), []byte(wwn), 0660); err != nil {
		return fmt.Errorf("error set nexus: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vhostLun, "iblock")); os.IsNotExist(err) {
		if err := os.Symlink(targetRoot, filepath.Join(vhostLun, "iblock")); err != nil {
			return fmt.Errorf("error create symlink: %v", err)
		}
	}
	return nil
}

func VHostDeleteIBlock(wwn string) error {
	vhostRoot := filepath.Join(tgtPath, "vhost", wwn, "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	if _, err := os.Stat(vhostLun); os.IsNotExist(err) {
		return fmt.Errorf("vHost do not exists for wwn %s: %s", wwn, err)
	}
	if err := os.Remove(filepath.Join(vhostLun, "iblock")); err != nil {
		return fmt.Errorf("error delete symlink: %v", err)
	}
	if err := os.RemoveAll(vhostLun); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	if err := os.RemoveAll(vhostRoot); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	if err := os.RemoveAll(filepath.Dir(vhostRoot)); err != nil {
		return fmt.Errorf("error delete lun: %v", err)
	}
	return nil
}

func TargetCreateIBlock(dev, tgtName, serial string) error {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	err := os.MkdirAll(targetRoot, os.ModeDir)
	if err != nil {
		return fmt.Errorf("cannot create fileio: %v", err)
	}
	if err := waitForFile(filepath.Join(targetRoot, "control")); err != nil {
		return fmt.Errorf("error waitForFile: %v", err)
	}
	controlCommand := fmt.Sprintf("udev_path=%s", dev)
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "control"), []byte(controlCommand), 0660); err != nil {
		return fmt.Errorf("error set control: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "wwn", "vpd_unit_serial"), []byte(serial), 0660); err != nil {
		return fmt.Errorf("error set vpd_unit_serial: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(targetRoot, "enable"), []byte("1"), 0660); err != nil {
		return fmt.Errorf("error set enable: %v", err)
	}
	return nil
}

// TargetDeleteIBlock - Delete iblock target
func TargetDeleteIBlock(tgtName string) error {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	if _, err := os.Stat(targetRoot); os.IsNotExist(err) {
		return fmt.Errorf("tgt do not exists for tgtName %s: %s", tgtName, err)
	}
	if err := os.RemoveAll(targetRoot); err != nil {
		return fmt.Errorf("error delete tgt: %v", err)
	}
	return nil
}

func GetSerialTarget(tgtName string) (string, error) {
	targetRoot := filepath.Join(iBlockPath, tgtName)
	//it returns something like "T10 VPD Unit Serial Number: 5001405043a8fbf4"
	serial, err := ioutil.ReadFile(filepath.Join(targetRoot, "wwn", "vpd_unit_serial"))
	if err != nil {
		return "", fmt.Errorf("GetSerialTarget for %s: %s", targetRoot, err)
	}
	parts := strings.Fields(strings.TrimSpace(string(serial)))
	if len(parts) == 0 {
		return "", fmt.Errorf("GetSerialTarget for %s: empty line", targetRoot)
	}
	return parts[len(parts)-1], nil
}

func IsVhostIblockExist(tgtName string) (bool, error) {
	serial, err := GetSerialTarget(tgtName)
	if err != nil {
		return false, fmt.Errorf("CheckVHostIBlock (%s): %v", tgtName, err)
	}

	vhostRoot := filepath.Join(tgtPath, "vhost", fmt.Sprintf("naa.%s", serial), "tpgt_1")
	vhostLun := filepath.Join(vhostRoot, "lun", "lun_0")
	if _, err := os.Stat(filepath.Join(vhostLun, "iblock")); err == nil {
		return true, nil
	}
	return false, nil
}

func GenerateNaaSerial() string {
	return fmt.Sprintf("%s%09x", naaPrefix, rand.Uint32())
}

func SetupVhost(devicePath, iblockId string) (string, error) {
	serial := GenerateNaaSerial()
	wwn := fmt.Sprintf("naa.%s", serial)
	err := TargetCreateIBlock(devicePath, iblockId, serial)
	if err != nil {
		return "", fmt.Errorf("TargetCreateFileIODev(%s, %s, %s): %v",
							  devicePath, iblockId, serial, err)
	}
	exists,err := IsVhostIblockExist(iblockId)
	if !exists {
		err = VHostCreateIBlock(iblockId, wwn)
		if err != nil {
			errString := fmt.Sprintf("VHostCreateIBlock: %v", err)
			err = VHostDeleteIBlock(wwn)
			if err != nil {
				errString = fmt.Sprintf("%s; VHostDeleteIBlock: %v",
					errString, err)
			}
			return "", fmt.Errorf("VHostCreateIBlock(%s, %s): %s",
				iblockId, wwn, errString)
		}
	}
	return wwn, nil
}
