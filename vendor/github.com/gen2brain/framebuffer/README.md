## framebuffer

**Note**: This is work in progress. Use at your own risk.

This library allows a Go application to draw arbitrary graphics to
the Linux Framebuffer. This allows one to create graphical applications
without the need for a monolithic display manager like X.

This is a pure Go implementation.

Because the framebuffer offers direct access to a chunk of memory mapped
pixel data, it is strongly advised to keep all actual drawing operations
confined to the thread that initialized the framebuffer.

The framebuffer is usually initialized to a specific display mode by the
kernel itself. While this library supplies the means to alter the current
display mode, this may not always have any effect as a driver can
choose to ignore your requested values. Besides that, it is generally
considered safer to use the external `fbset` command for this purpose.
Video modes for the framebuffer require very precise timing values to
be supplied along with any desired resolution. Doing this incorrectly
can damage the display.

`fbset` comes with a set of default modes which are stored in the file
`/etc/fb.modes`. We read this file and extract the set of
video modes from it. These modes each have a name by which they can
be identified. When supplying a new mode to this package, it should
come in the form of this name. For example: `"1600x1200-76"`.

New video modes can be added to the `/etc/fb.modes` file.

The framebuffer obscures the terminal, so any debug or error data
written to `stdout` and/or `stderr`, will not be visible while it
is running. To get access to this data, pipe their outputs to a file:

	./myapp 1> stdout.log 2>error.log


### Known issues

* Running a program which writes to the Framebuffer, may fail with
  a permission error. This is likely because your current user is not
  part of the `video` group. Add the user to this group and all should
  be well.

* This library draws directly the raw framebuffer. This means that it
  may interfere with other applications doing the same thing. This
  includes X, if it is running. While we take every effort to appropriately
  handle switching between the various applications, this should always
  be kept in mind.

* There may be ioctl errors when trying to run a program in a terminal
  emulator. This happens because the API requires a real tty. Ideally
  this should be fixed at some point.


### Usage

    go get github.com/jteeuwen/framebuffer


### License

Unless otherwise stated, all of the work in this project is subject to a
1-clause BSD license. Its contents can be found in the enclosed LICENSE file.

