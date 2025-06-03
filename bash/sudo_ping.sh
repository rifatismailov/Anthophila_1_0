#!/usr/bin/expect
set timeout -1

# Отримуємо пароль з аргументу
set password [lindex $argv 0]

# Виконуємо команду з sudo, передаючи пароль через stdin
spawn sudo -S ping -c 4 8.8.8.8

# Очікуємо запит пароля
expect "Password:"

# Відправляємо пароль
send "$password\r"

# Очікуємо завершення
expect eof
