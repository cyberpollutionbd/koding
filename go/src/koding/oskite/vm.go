// +build linux

package oskite

import (
	"fmt"
	"koding/tools/dnode"
	"koding/tools/kite"
	"koding/virt"
	"time"

	"labix.org/v2/mgo/bson"
)

func vmStart(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}

	done := make(chan struct{}, 1)
	prepareQueue <- &QueueJob{
		msg: "vm.Start" + channel.CorrelationName,
		f: func() string {

			if err := vos.VM.Start(); err != nil {
				panic(err)
			}

			// wait until network is up
			if err := vos.VM.WaitForNetwork(time.Second * 5); err != nil {
				panic(err)
			}

			done <- struct{}{}
			return fmt.Sprintf("vm.Start %s", channel.CorrelationName)
		},
	}

	<-done

	return true, nil
}

func vmShutdown(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}

	done := make(chan struct{}, 1)
	prepareQueue <- &QueueJob{
		msg: "vm.Shutdown" + channel.CorrelationName,
		f: func() string {

			if err := vos.VM.Shutdown(); err != nil {
				panic(err)
			}

			done <- struct{}{}
			return fmt.Sprintf("vm.Shutdown %s", channel.CorrelationName)
		},
	}

	<-done
	return true, nil
}

func vmUnprepare(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}

	if err := vos.VM.Unprepare(); err != nil {
		return nil, err
	}

	return true, nil
}

func vmStop(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}
	if err := vos.VM.Stop(); err != nil {
		panic(err)
	}
	return true, nil
}

func vmReinitialize(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}
	vos.VM.Prepare(true, log.Warning)
	if err := vos.VM.Start(); err != nil {
		panic(err)
	}
	return true, nil
}

func vmInfo(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	info := channel.KiteData.(*VMInfo)
	info.State = vos.VM.GetState()
	return info, nil
}

func vmResizeDisk(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}
	return true, vos.VM.ResizeRBD()
}

func vmCreateSnaphost(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	if !vos.Permissions.Sudo {
		return nil, &kite.PermissionError{}
	}

	snippetId := bson.NewObjectId().Hex()
	if err := vos.VM.CreateConsistentSnapshot(snippetId); err != nil {
		return nil, err
	}

	return snippetId, nil
}

func spawnFunc(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	var command []string
	if args.Unmarshal(&command) != nil {
		return nil, &kite.ArgumentError{Expected: "[array of strings]"}
	}
	return vos.VM.AttachCommand(vos.User.Uid, "", command...).CombinedOutput()
}

func execFunc(args *dnode.Partial, channel *kite.Channel, vos *virt.VOS) (interface{}, error) {
	var line string
	if args.Unmarshal(&line) != nil {
		return nil, &kite.ArgumentError{Expected: "[string]"}
	}
	return vos.VM.AttachCommand(vos.User.Uid, "", "/bin/bash", "-c", line).CombinedOutput()
}
