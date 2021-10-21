#!/bin/sh

(echo no; echo no; echo no; echo yes; echo http://lotus:1234/rpc/v0; echo $(cat ~/.lotus/token); echo no; echo yes; echo 10; echo no) | fcr init
IPFS_LOGGING=info fcr daemon
