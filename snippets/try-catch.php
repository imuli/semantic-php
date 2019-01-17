<?php
   try {
      throw new Exception('blah');
   } catch (Exception $e) {
      echo $e->getMessage();
   }
?>
