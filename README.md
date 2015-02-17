# Baci - Build an ACI.

Baci is an ACI builder.

## How is it implemented?
The build process is done inside the container (it's an ACI that builds other ACIs!).

I'm not already sure if this was good choice but it was the easiest.
Maybe in the future this can move from being an ACI to become an extension to the rocket stage1 or just an external program that uses rkt (this will need rocket to implement some additional features).

By now some issues to deal with are known and other can arise:
 * There's the need to deal with the other filesystems mounted by the ACE (now and in the future) like /dev, /proc, /sys etc...
 * The base image needs to be extracted by the baci's aci.


## Why are you using a Dockerfile?
This isn't a replacement for `docker build` and I'm pretty sure that trying to create an ACI using a Dockerfile that's 100% identical to the images created by docker is probably impossible.

I started with the Dockerfile build language as I needed something ready to test.
By now base images (FROM) are downloaded and converted using https://github.com/appc/docker2aci (or better, the rocket's cas, which uses docker2aci, is used to get them). In future also other (non docker) aci should be used as base images.

## What about other build languages?
I think that a build language like a Dockerfile is fundamental to create containers in an easy and (semi)reproducible way.

By now, just for the reasons explained above an initial DockerBuilder has been developed.

I'm hoping that a standard container build language will be defined/created (peraphs something with a better separation between the build directives and the container definition, like exec program, volumes etc...).


## Build
```
git clone https://github.com/sgotti/baci
cd baci
./build
```

All the needed files are created inside `./bin`

Note: the build scripts tries to get a local copy of `xz` and the needed libraries (it's needed to extract tar.xz files). If xz is not found then a warning is reported. This is tested on a fedora system (to install xz use `{yum|dnf} install xz`.

## Examples

### Base fedora image
`git clone https://github.com/fedora-cloud/docker-brew-fedora`

`cd docker-brew-fedora`

Right now baci needs to be run as root.

`$BACIBIN/baci --rocketpath $RKTBIN/rkt -o $OUTDIR/fedora.aci .`

If everything goes ok you'll find your aci in $OUTDIR/fedora.aci and you can run it with `$RKTBIN/rkt run $OUTDIR/fedora.aci` (as the exec cmd is `/bin/bash`, now it will just exit)


### Something more complex: fedora based postgresql
`git clone https://github/docker-library/fedora-cloud`

`$BACIBIN/baci --rocketpath $RKTBIN/rkt -o $OUTDIR/postgre.aci ./fedora-cloud/Fedora-Dockerfiles/postgres/`

When finished:

`$RKTBIN/rkt run $OUTDIR/postgre.aci`

Access your rocketized postgresql.



## TODO

* Make this todos github's issues...
* Use acirenderer to render ACIs with dependencies.
* Use acibuilder's diff builder to create only ACIs with differences from the base image. 

