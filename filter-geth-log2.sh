cat /Users/ia/gocode/src/github.com/rotblauer/piper/in.txt \
| grep -v -E '(Removed|Added|mismatch)' \
|grep -v -E '(Removing)' \
|grep -v -E '(Accepted|inbound|Quality)' \
|grep -v -E '(requested disconnect|192000)' \
| sed '/peer connected/d' \
|sed '/dyn dial/d' \
|sed '/EOF/d' \
|sed '/shutting down/d' \
|grep -v -E '(out fork-check)' \
|grep -v busy \
