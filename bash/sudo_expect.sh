#!/usr/bin/expect
set timeout -1

# Запускаємо команду з привілеями sudo
spawn sudo ping -c 4 8.8.8.8

# Очікуємо запит пароля
expect "Password:"

# Відправляємо пароль
send "yorPass\r"

# Далі взаємодіємо з процесом
interact
