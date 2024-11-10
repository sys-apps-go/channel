Scan a directory, grep for pattern in files using worker threads with buffered and unbuffered channels and also channel array:
Verifying the following:
1. Wait and Wakeup and context switch for unbuffered channel worker threads
2. Buffered channel with size of 1024, with less context switch and wait and wakeup
3. Channel array for less context switch for sender and receiver

time go run fsgrep_buffered_channel.go main ~/work -n 32 -k 32  | wc -l
time go run fsgrep_unbuffered_channel.go main ~/work -n 32 -k 32  | wc -l
time go run fsgrep_buffered_channel_array.go main ~/work -n 32 -k 32  | wc -l
time go run fsgrep_unbuffered_channel_array.go main ~/work -n 32 -k 32  | wc -l
time go run fsgrep_mq.go main ~/work -n 32 -k 32  | wc -l
