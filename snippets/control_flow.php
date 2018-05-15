<?php

if($hello){
	echo "Hello World";
} else {
	echo "Goodbye";
}

foreach($loc as $l){
	// do something with $l
}

for($i = 0; $i < 10; $i++){
	echo $i;
}

while(false){
	echo "Not seeing this";
}

do {
	echo "Once";
} while(false);

switch($x){
case 2:
	echo "It's a 2!";
	break;
}

if($ut):
	echo $ut;
endif;

for($i = 0; $i < 10; $i++):
	echo $i * $i;
endfor;

switch($x):
case 1:
	echo "It's a 1!";
	break;
endswitch;

foreach($loc as $alt):
	blah();
endforeach;

while(false):
	echo "Nothing Here";
endwhile;

$seriously ? 'weird' : 'ok then';
