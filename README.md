# grpctunnel-demo

This demos concurrency issues with https://github.com/jhump/grpctunnel

Steps:
- Build client & server binaries like so: `make build`
- Replace `src_data.bin` file with your test data. I use a binary file of about ~ 2GB
- Run `network_server` in terminal 1
- Run `network_client` in terminal 2
- To initiate data transfer, signal `network_server` with SIGHUP like so: `pkill -SIGHUP network_server`
- The files will be saved as: `dst_data_at_0.bin` `dst_data_at_1.bin` ...

To reproduce:
- Initiate two file transfers by issuing `SIGHUP` to `network_server` twice

Expected:
- The `network_client` shouild create exact copies of `src_data.bin` like so: `dst_data_at_0.bin` `dst_data_at_1.bin`

Actual:
- A single complete copy is created. The second transfer stalls resulting in an incomplete transfer.
