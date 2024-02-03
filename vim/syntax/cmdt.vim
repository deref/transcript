" Vim syntax file
" Language: Command Transcripts
" Maintainer: Brandon Bloom
" Latest Revision: 3 February 2024

if exists("b:current_syntax")
  finish
endif

syn match cmdtComment '# [^\n]*'
syn match cmdtDirective '% [^\n]*'
syn match cmdtCommand '\$ [^\n]*'
syn match cmdtStdout '1 [^\n]*'
syn match cmdtStderr '2 [^\n]*'
syn match cmdtExitCode '? [^\n]*'

hi def link cmdtComment    Comment
hi def link cmdtDirective  Special
hi def link cmdtCommand    Statement
hi def link cmdtStdout     Identifier
hi def link cmdtStderr     WarningMsg
hi def link cmdtExitCode   Number
