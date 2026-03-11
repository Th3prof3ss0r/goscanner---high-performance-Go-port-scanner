# goscanner high-performance-Go-port-scanner

# Dentro de C:\goscanner
go run . 127.0.0.1 -p 80,443,22

# Top 100 portas no localhost
go run . 127.0.0.1 -p top100

# Portas específicas com detecção de versão
go run . 127.0.0.1 -p 22,80,443,3306 -sV

# CIDR com saída em tabela
go run . 192.168.1.0/24 -p 80,443 --open

# Compilar primeiro, depois executar (mais rápido)
go build -o goscanner.exe .
.\goscanner.exe 127.0.0.1 -p 1-1024
