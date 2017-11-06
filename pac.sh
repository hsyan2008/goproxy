#/bin/bash
rm -rf abc.pac
aria2c -q https://pac.itzmx.com/abc.pac
echo "hosts = [" > pac.toml
grep '": 1' abc.pac  | awk -F\" '{print " \""$2"\","}' >> pac.toml

#自定义的域名加到这里
echo " \"google.com\"," >> pac.toml

echo "]" >> pac.toml
