// +build !windows

package service

import (
	"flag"
	"fmt"
	"github.com/ErikDubbelboer/gspt"
	"gopkg.in/hlandau/service.v1/daemon"
	"gopkg.in/hlandau/service.v1/daemon/pidfile"
	"gopkg.in/hlandau/service.v1/passwd"
	"gopkg.in/hlandau/service.v1/sdnotify"
	"os"
	"strconv"
)

// This will always point to a path which the platform guarantees is an empty
// directory. You can use it as your default chroot path if your service doesn't
// access the filesystem after it's started.
//
// On Linux, the FHS provides that "/var/empty" is an empty directory, so it
// points to that.
var EmptyChrootPath = daemon.EmptyChrootPath

var (
	uidFlag        = fs.String("uid", "", "UID to run as (default: don't drop privileges)")
	_uidFlag       = flag.String("uid", "", "UID to run as (default: don't drop privileges)")
	gidFlag        = fs.String("gid", "", "GID to run as (default: don't drop privileges)")
	_gidFlag       = flag.String("gid", "", "GID to run as (default: don't drop privileges)")
	daemonizeFlag  = fs.Bool("daemon", false, "Run as daemon? (doesn't fork)")
	_daemonizeFlag = flag.Bool("daemon", false, "Run as daemon? (doesn't fork)")
	chrootFlag     = fs.String("chroot", "", "Chroot to a directory (must set UID, GID) (\"/\" disables)")
	_chrootFlag    = flag.String("chroot", "", "Chroot to a directory (must set UID, GID) (\"/\" disables)")
	pidfileFlag    = fs.String("pidfile", "", "Write PID to file with given filename and hold a write lock")
	_pidfileFlag   = flag.String("pidfile", "", "Write PID to file with given filename and hold a write lock")
	dropprivsFlag  = fs.Bool("dropprivs", true, "Drop privileges?")
	_dropprivsFlag = flag.Bool("dropprivs", true, "Drop privileges?")
	forkFlag       = fs.Bool("fork", false, "Fork? (implies -daemon)")
	_forkFlag      = flag.Bool("fork", false, "Fork? (implies -daemon)")
)

func systemdUpdateStatus(status string) error {
	return sdnotify.Send(status)
}

func setproctitle(status string) error {
	gspt.SetProcTitle(status)
	return nil
}

func (info *Info) serviceMain() error {
	err := daemon.Init()
	if err != nil {
		return err
	}

	err = systemdUpdateStatus("\n")
	if err == nil {
		info.systemd = true
	}

	if *pidfileFlag != "" {
		info.pidFileName = *pidfileFlag

		err = info.openPIDFile()
		if err != nil {
			return err
		}

		defer info.closePIDFile()
	}

	return info.runInteractively()
}

func (info *Info) openPIDFile() error {
	return pidfile.OpenPIDFile(info.pidFileName)
}

func (info *Info) closePIDFile() {
	pidfile.ClosePIDFile()
}

func (h *ihandler) DropPrivileges() error {
	if h.dropped {
		return nil
	}

	if *forkFlag {
		isParent, err := daemon.Fork()
		if err != nil {
			return err
		}

		if isParent {
			os.Exit(0)
		}

		*daemonizeFlag = true
	}

	if *daemonizeFlag || h.info.systemd {
		err := daemon.Daemonize()
		if err != nil {
			return err
		}
	}

	if *uidFlag != "" && *gidFlag == "" {
		gid, err := passwd.GetGIDForUID(*uidFlag)
		if err != nil {
			return err
		}
		*gidFlag = strconv.FormatInt(int64(gid), 10)
	}

	if h.info.DefaultChroot == "" {
		h.info.DefaultChroot = "/"
	}

	chrootPath := *chrootFlag
	if chrootPath == "" {
		chrootPath = h.info.DefaultChroot
	}

	uid := -1
	gid := -1
	if *uidFlag != "" {
		var err error
		uid, err = passwd.ParseUID(*uidFlag)
		if err != nil {
			return err
		}

		gid, err = passwd.ParseGID(*gidFlag)
		if err != nil {
			return err
		}
	}

	if *dropprivsFlag {
		chrootErr, err := daemon.DropPrivileges(uid, gid, chrootPath)
		if err != nil {
			return fmt.Errorf("Failed to drop privileges: %v", err)
		}
		if chrootErr != nil && *chrootFlag != "" && *chrootFlag != "/" {
			return fmt.Errorf("Failed to chroot: %v", chrootErr)
		}
	} else if *chrootFlag != "" && *chrootFlag != "/" {
		return fmt.Errorf("Must set dropprivs to use chroot")
	}

	if !h.info.AllowRoot && daemon.IsRoot() {
		return fmt.Errorf("Daemon must not run as root or with capabilities; run as non-root user or use -uid")
	}

	h.dropped = true
	return nil
}

// © 2015 Hugo Landau <hlandau@devever.net>  ISC License
