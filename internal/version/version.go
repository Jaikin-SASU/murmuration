// Package version expose la version du binaire murmur.
package version

// Version est injectée au build via -ldflags "-X .../version.Version=vX.Y.Z".
var Version = "v0.1.0-dev"
