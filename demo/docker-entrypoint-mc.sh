#!/bin/sh

ip=$(ping mc -c 1 | grep "PING mc" | cut -d "(" -f2 | cut -d ")" -f1)
(echo no; echo no; echo no; echo yes; echo /ip4/$ip/tcp/9010/p2p/12D3KooWPRE8jwmDG6vjxXETqUG2PVMWZ12LuWFYTpcn7QSudJ92; echo no; echo no) | fcr init --debug
IPFS_LOGGING=info fcr daemon
