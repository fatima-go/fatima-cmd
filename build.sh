#!/bin/sh

Usage()
{
   # Display Usage
   echo "fatima command binary build script"
   echo
   echo "Syntax: build.sh [-o|a|d]"
   echo "options:"
   echo "o     GOOS. e.g) linux darwin"
   echo "a     GOARCH e.g) amd64 arm64"
   echo "d     install directory path"
   echo
}

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
    -o|--os)
      OS="$2"
      shift # past argument
      shift # past value
      ;;
    -a|--arch)
      ARCH="$2"
      shift # past argument
      shift # past value
      ;;
    -d|--dir)
      INSTALL_DIR="$2"
      shift # past argument
      shift # past value
      ;;
    --default)
      DEFAULT=YES
      shift # past argument
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      POSITIONAL_ARGS+=("$1") # save positional arg
      shift # past argument
      ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}" # restore positional parameters

if [ -z "$OS" ]
then
      echo "OS is empty\n"
      Usage
      exit 1
fi

if [ -z "$ARCH" ]
then
      echo "ARCH is empty\n"
      Usage
      exit 1
fi

if [ -z "$INSTALL_DIR" ]
then
      INSTALL_DIR=$GOPATH/pkg/${OS}_${ARCH}
fi

echo "install dir : ${INSTALL_DIR}"

base_dir=`pwd`
programs=(lcslack lcproc rocontext roupdate roclip rocron rodeploy roclric rohis rodis rolog ropack roproc lcps rostart rostop lcha startro stopro)

for pgm in ${programs[@]}; do
	dir=${base_dir}"/cmd/"${pgm}
	echo "installing ${pgm}"
	cd ${dir}
	GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o ${INSTALL_DIR}/${pgm}
done

exit 0
