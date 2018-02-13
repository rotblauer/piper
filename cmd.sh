cat /Users/ia/gocode/src/github.com/rotblauer/piper/in.txt \
|grep -v -E '(Quality|192000|Subprotocol|peer connected)' \
|grep -v -E '(mismatch|Genesis|shutting)' \
|grep -v -E '(Accepted|Removed|Added)' \
|sed '1,4d' \
