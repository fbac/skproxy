 #!/bin/bash
echo "hello" | nc -4 localhost 5001 &
sleep 0.5
echo "hello" | nc -4 localhost 5200 &
sleep 0.5
echo "hello" | nc -4 localhost 5300 &
sleep 0.5
echo "hello" | nc -4 localhost 5400 &
sleep 0.5
echo "hello" | nc -4 localhost 6001 &
sleep 0.5
echo "hello" | nc -4 localhost 6200 &
sleep 0.5
echo "hello" | nc -4 localhost 6300 &
sleep 0.5
echo "hello" | nc -4 localhost 6400 &
sleep 0.5
echo "hello" | nc -4 localhost 7001 &
sleep 0.5
echo "hello" | nc -4 localhost 7200 &
sleep 0.5
echo "hello" | nc -4 localhost 7300 &
sleep 0.5
echo "hello" | nc -4 localhost 7400 &
sleep 0.5