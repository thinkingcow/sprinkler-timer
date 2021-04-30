# sprinkler-timer
Sprinkler timer using the SequentMicrosystems stackable 8-relay board(s)

This code is/was reverse engineered by running the 8relay binary from SequentMicro with
strace -e open,read,write,ioctl 8relay ...
