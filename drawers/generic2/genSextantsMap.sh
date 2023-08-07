#!/usr/bin/env bash

# U1FB00.pdf

binary(){
  i=0
  printf '0b%06d' "$(
    echo "obase=2;$((i"$(<<<"$1" sed 's,.,|1<<(&-1),g')"))" | \
    bc
  )"
}

gen(){
  echo 'package generic2'
  echo
  echo 'var sextants = map[uint8]rune{'
  {
    while read -r line; do
      echo $'\t'"$(
        binary "$(
          <<<"$line" sed 's,.*-,,;y,123456,142536,'
        )"
      ): 0x$(
        <<<"$line" sed 's| |,&// |'
      )"
    done <sextants
	  echo $'\t'"0b000000: 0x0020, // ' ' SPACE"
	  echo $'\t''0b111111: 0x2588, // █ FULL BLOCK'
  	echo $'\t''0b111000: 0x258C, // ▌ LEFT HALF BLOCK'
	  echo $'\t''0b000111: 0x2590, // ▐ RIGHT HALF BLOCK'
  } | \
  sort
  echo '}'
}

Main(){
  filename='sextants.go'
  gen >"$filename"
  chmod 755 "$filename"
  go fmt "$filename" >/dev/null
}

Main "$@"
