:] �1(�2(�3(�4('	�  � �  ���Usage: RET [1|2|3|4]
  1 - Exit via 'JP 0x0000'.
  2 - Exit via 'RST 0' instruction.
  3 - Exit via 'RET'.
  4 - Exit via 'P_TERMCPM' syscall.
$