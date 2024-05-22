fsnotify is a Go library to provide cross-platform filesystem notifications on
Windows, Linux, macOS, and BSD systems.

Go 1.16 or newer is required; the full documentation is at
https://pkg.go.dev/github.com/fsnotify/fsnotify

**It's best to read the documentation at pkg.go.dev, as it's pinned to the last
released version, whereas this README is for the last development version which
may include additions/changes.**

---

```console
go get -u golang.org/x/sys/...
```

| Adapter               | OS             | Status                                                       |
| --------------------- | ---------------| -------------------------------------------------------------|
| inotify               | Linux 2.6.32+  | Supported                                                    |
| kqueue                | BSD, macOS     | Supported                                                    |
| ReadDirectoryChangesW | Windows        | Supported                                                    |
| FSEvents              | macOS          | [Planned](https://github.com/fsnotify/fsnotify/issues/11)    |
| FEN                   | Solaris 11     | [In Progress](https://github.com/fsnotify/fsnotify/pull/371) |
| fanotify              | Linux 5.9+     | [Maybe](https://github.com/fsnotify/fsnotify/issues/114)     |
| USN Journals          | Windows        | [Maybe](https://github.com/fsnotify/fsnotify/issues/53)      |
| Polling               | *All*          | [Maybe](https://github.com/fsnotify/fsnotify/issues/9)       |

Linux and macOS should include Android and iOS, but these are currently untested.

Please see [the documentation](https://godoc.org/github.com/fsnotify/fsnotify) and consult the [FAQ](#faq) for usage information.

## API stability

fsnotify is a fork of [howeyc/fsnotify](https://godoc.org/github.com/howeyc/fsnotify) with a new API as of v1.0. The API is based on [this design document](http://goo.gl/MrYxyA). 

All [releases](https://github.com/fsnotify/fsnotify/releases) are tagged based on [Semantic Versioning](http://semver.org/). Further API changes are [planned](https://github.com/fsnotify/fsnotify/milestones), and will be tagged with a new major revision number.

Go 1.6 supports dependencies located in the `vendor/` folder. Unless you are creating a library, it is recommended that you copy fsnotify into `vendor/github.com/fsnotify/fsnotify` within your project, and likewise for `golang.org/x/sys`.

## Usage

```go
package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("/tmp/foo")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
```

## Contributing

Please refer to [CONTRIBUTING][] before opening an issue or pull request.

FAQ
---
### Will a file still be watched when it's moved to another directory?
No, not unless you are watching the location it was moved to.

### Are subdirectories watched too?
No, you must add watches for any directory you want to watch (a recursive
watcher is on the roadmap: [#18]).

**When a file is moved to another directory is it still being watched?**

No (it shouldn't be, unless you are watching where it was moved to).

**When I watch a directory, are all subdirectories watched as well?**

No, you must add watches for any directory you want to watch (a recursive watcher is on the roadmap [#18][]).

**Do I have to watch the Error and Event channels in a separate goroutine?**

As of now, yes. Looking into making this single-thread friendly (see [howeyc #7][#7])

**Why am I receiving multiple events for the same file on OS X?**

Spotlight indexing on OS X can result in multiple events (see [howeyc #62][#62]). A temporary workaround is to add your folder(s) to the *Spotlight Privacy settings* until we have a native FSEvents implementation (see [#11][]).

**How many files can be watched at once?**

There are OS-specific limits as to how many watches can be created:
* Linux: /proc/sys/fs/inotify/max_user_watches contains the limit, reaching this limit results in a "no space left on device" error.
* BSD / OSX: sysctl variables "kern.maxfiles" and "kern.maxfilesperproc", reaching these limits results in a "too many open files" error.

**Why don't notifications work with NFS filesystems or filesystem in userspace (FUSE)?**

fsnotify requires support from underlying OS to work. The current NFS protocol does not provide network level support for file notifications.

[#62]: https://github.com/howeyc/fsnotify/issues/62
[#18]: https://github.com/fsnotify/fsnotify/issues/18

### Do I have to watch the Error and Event channels in a goroutine?
As of now, yes (you can read both channels in the same goroutine using `select`,
you don't need a separate goroutine for both channels; see the example).

### Why don't notifications work with NFS, SMB, FUSE, /proc, or /sys?
fsnotify requires support from underlying OS to work. The current NFS and SMB
protocols does not provide network level support for file notifications, and
neither do the /proc and /sys virtual filesystems.

This could be fixed with a polling watcher ([#9]), but it's not yet implemented.

[#9]: https://github.com/fsnotify/fsnotify/issues/9

Platform-specific notes
-----------------------
### Linux
When a file is removed a REMOVE event won't be emitted until all file
descriptors are closed; it will emit a CHMOD instead:

    fp := os.Open("file")
    os.Remove("file")        // CHMOD
    fp.Close()               // REMOVE

This is the event that inotify sends, so not much can be changed about this.

The `fs.inotify.max_user_watches` sysctl variable specifies the upper limit for
the number of watches per user, and `fs.inotify.max_user_instances` specifies
the maximum number of inotify instances per user. Every Watcher you create is an
"instance", and every path you add is a "watch".

These are also exposed in `/proc` as `/proc/sys/fs/inotify/max_user_watches` and
`/proc/sys/fs/inotify/max_user_instances`

To increase them you can use `sysctl` or write the value to proc file:

    # The default values on Linux 5.18
    sysctl fs.inotify.max_user_watches=124983
    sysctl fs.inotify.max_user_instances=128

To make the changes persist on reboot edit `/etc/sysctl.conf` or
`/usr/lib/sysctl.d/50-default.conf` (details differ per Linux distro; check your
distro's documentation):

    fs.inotify.max_user_watches=124983
    fs.inotify.max_user_instances=128

Reaching the limit will result in a "no space left on device" or "too many open
files" error.

### kqueue (macOS, all BSD systems)
kqueue requires opening a file descriptor for every file that's being watched;
so if you're watching a directory with five files then that's six file
descriptors. You will run in to your system's "max open files" limit faster on
these platforms.

The sysctl variables `kern.maxfiles` and `kern.maxfilesperproc` can be used to
control the maximum number of open files.

### macOS
Spotlight indexing on macOS can result in multiple events (see [#15]). A temporary
workaround is to add your folder(s) to the *Spotlight Privacy settings* until we
have a native FSEvents implementation (see [#11]).

[#11]: https://github.com/fsnotify/fsnotify/issues/11
[#15]: https://github.com/fsnotify/fsnotify/issues/15
