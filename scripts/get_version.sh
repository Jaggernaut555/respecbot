# Get the version.
version=`git describe --tags --long`
# Write out the package.
cat << EOF > version.go
package main

//go:generate bash ./scripts/get_version.sh
var Version = "$version"
EOF