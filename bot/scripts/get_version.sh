# Get the version.
version=`git describe --tags --long`
# Write out the package.
cat << EOF > version.go
package bot

//go:generate bash ./scripts/get_version.sh

//Version version of the project
var Version = "$version"
EOF