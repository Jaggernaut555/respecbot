#!/bin/bash
isExistApp=`pgrep respecbot`
if [[ -n  $isExistApp ]]; then
   supervisorctl stop respecbot
fi
