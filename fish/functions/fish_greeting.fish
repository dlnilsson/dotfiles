function fish_greeting

    set_color yellow
    #printf '%s \n%s\n%s \n' $__fish_prompt_hostname (date '+%A - %H:%M') (istats battery charge) (istats cpu temp)
    set_color normal
end
