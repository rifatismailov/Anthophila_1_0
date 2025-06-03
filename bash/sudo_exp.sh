#!/usr/bin/expect -f
set timeout -1

# Отримуємо аргументи
set password [lindex $argv 0]
set cmd [join [lrange $argv 1 end] " "]

# Запускаємо команду напряму через spawn (без bash -c)
spawn sudo -S {*}$cmd

# Чекаємо на пароль
expect {
    "Password:" {
        send "$password\r"
        exp_continue
    }
    eof
}
