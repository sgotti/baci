## ACIBuilder library ##

An ACIBuilder library that will create an ACI.

Pluggable Builders can be used (they just need to implement the ACIBuilder interface that is composed by only the Build() function)

At the moment two builders are provided:
 * A simple builder that will build an ACI given an ImageManifest and a filesystem path.
 * A diff builder that will build an ACI given an ImageManifest a base ACI (exploded) and the new ACI (exploded). The generated ACI will contain only the differences from the base ACI. If there are deleted files from the base ACI, the imagemanifest will be augmented with a pathWhiteList containing all the ACI's files

In future additional builders will be available (for example an overlayfs builder).



