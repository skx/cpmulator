	� ��`͟�� � E	� !���� ��w#��	� �	� !�~��~_� ��#��	� ��	� !���� ��w#�v	� !�~��~_� ��#��	� ��	� !	� �� �$	� �� (��'	� ��q ��	� �*	� !�>w��
� �	� l	� !�~G#��~_� ����	� �Simple input-test program, by Steve.

$
C_READ Test:
  This test allows you to enter FIVE characters, one by one.
  The characters SHOULD be echoed as you type them.
$  Test complete - you entered '$'.
$
A_READ Test:
  This test allows you to enter FIVE characters, one by one.
  The characters should NOT be echoed as you type them.
$  Test complete - you entered '$'.
$
C_RAWIO Test:
  This uses polling to read characters.
  Echo should NOT be enabled.
  Press 'q' to proceed/complete this test.
$x$X$+$
C_READSTRING Test:
  Enter a string, terminated by newline..
$  Test complete - you entered '$'.
$