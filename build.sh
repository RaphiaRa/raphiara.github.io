#rm -r out
cd src

go run . $1
cd ..
cp -R css out/
