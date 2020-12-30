launcher
========
One of the things that has always annoyed me of all those applications with a remote control is that in order to control them remotely,
you have to start them first, and that ability is never on offer.

enter launcher
--------------
Here's a little HTTP server that takes care of that.

It reads a JSON configuration file with the apps you want to monitor, and produces a simple HTML interface, to start them, stop them, or simply check their status.

It's best placed in your desktop startup chain (eg for KDE system settings / Start and Stop / Autostart, create a script).

building
--------

> GOPATH=`pwd` go build ./...

running
-------

> launcher &

hints
-----
New applications can be added to launcher.json, the format is pretty much self explanatory.

It's best to start each app using the shell and running them in the background, as in launcher.json, if not, killing them will result in zombie processes still
attached to the launcher, which them breaks monitoring their status, and restarting them.

I see no reason why you should not be able to use this little tool on Windows or OSX as well.

You can disable security and change the port from launcher.json as well.

The HTML style can be changed via resources/launcher.css.

acknowledgements
----------------
The icons have been taken from http://freesvg.org. I understand they are in the public domain.
