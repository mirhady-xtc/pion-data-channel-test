go build -o server server.go
./server -data_channel_count 1 -experiment_count 1 -message_interval 1 -message_size 800 -num_messages 10000 -path_count 1 -use_wss 0 -use_interface_filter 0
