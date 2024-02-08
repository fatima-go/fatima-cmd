#!/bin/sh

#
# Copyright 2023 github.com/fatima-go
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# @project fatima-core
# @author jin
# @date 23. 4. 14. 오후 5:07
#

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
programs=(lcslack lcproc lccrypto rocontext roupdate roclip rocron rodeploy roclric rohis rodis rolog ropack roproc lcps rostart rostop lcha startro stopro)

for pgm in ${programs[@]}; do
	dir=${base_dir}"/cmd/"${pgm}
	echo "installing ${pgm}"
	cd ${dir}
	GOOS=$OS GOARCH=$ARCH go build -ldflags="-s -w" -o ${INSTALL_DIR}/${pgm}
done

exit 0
