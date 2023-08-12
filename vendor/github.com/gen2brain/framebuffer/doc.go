// This file is subject to a 1-clause BSD license.
// Its contents can be found in the enclosed LICENSE file.

/*
This library allows a Go application to draw arbitrary graphics to
the Linux Framebuffer. This allows one to create graphical applications
without the need for a monolithic display manager like X.

Because the framebuffer offers direct access to a chunk of memory mapped
pixel data, it is strongly advised to keep all actual drawing operations
confined to the thread that initialized the framebuffer.
*/
package framebuffer
