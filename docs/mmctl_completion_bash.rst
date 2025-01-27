.. _mmctl_completion_bash:

mmctl completion bash
---------------------

Generates the bash autocompletion scripts

Synopsis
~~~~~~~~


To load completion, run

. <(mmctl completion bash)

To configure your bash shell to load completions for each session, add the above line to your ~/.bashrc


::

  mmctl completion bash [flags]

Options
~~~~~~~

::

  -h, --help   help for bash

Options inherited from parent commands
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

      --config string                path to the configuration file (default "$XDG_CONFIG_HOME/mmctl/config")
      --insecure-sha1-intermediate   allows to use insecure TLS protocols, such as SHA-1
      --insecure-tls-version         allows to use TLS versions 1.0 and 1.1
      --json                         the output format will be in json format
      --local                        allows communicating with the server through a unix socket
      --quiet                        prevent mmctl to generate output for the commands
      --strict                       will only run commands if the mmctl version matches the server one
      --suppress-warnings            disables printing warning messages

SEE ALSO
~~~~~~~~

* `mmctl completion <mmctl_completion.rst>`_ 	 - Generates autocompletion scripts for bash and zsh

