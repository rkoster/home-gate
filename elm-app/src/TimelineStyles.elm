module TimelineStyles exposing (intervalBlock, legendActive, legendInactive, tooltip, hourBlock)

import Css exposing (..)

intervalBlock : Bool -> Css.Style
intervalBlock isActive =
    let
        baseColor = if isActive then hex "30c750" else hex "e5e5e5"
    in
    Css.batch
        [ display inlineBlock
        , width (px 18)
        , height (px 24)
        , marginRight (px 1)
        , backgroundColor baseColor
        , borderRadius (px 3)
        , Css.property "transition" "transform 0.17s cubic-bezier(.25,.8,.50,1)"
        , hover
            [ transform (scaleY 1.17)
            , boxShadow4 (px 0) (px 0) (px 8) (rgba 16 185 129 0.4)
            ]
        , transform (scaleY (if isActive then 0.98 else 0.9))
        , boxShadow4 (px 0) (px 0) (px 8) (if isActive then (rgba 16 185 129 0.19) else (rgba 0 0 0 0))
        , cursor pointer
        ]

legendActive : Css.Style
legendActive =
    Css.batch
        [ display inlineBlock
        , width (px 16)
        , height (px 16)
        , backgroundColor (hex "30c750")
        , borderRadius (px 2)
        , marginRight (px 4)
        ]

legendInactive : Css.Style
legendInactive =
    Css.batch
        [ display inlineBlock
        , width (px 16)
        , height (px 16)
        , backgroundColor (hex "bbbbbb")
        , borderRadius (px 2)
        , marginRight (px 4)
        ]

tooltip : Css.Style
tooltip =
    Css.batch
        [ position absolute
        , Css.property "bottom" "110%"
        , left (pct 50)
        , Css.property "transform" "translateX(-50%)"
        , backgroundColor (hex "111827")
        , color (hex "ffffff")
        , fontSize (px 12)
        , padding2 (px 6) (px 12)
        , borderRadius (px 4)
        , whiteSpace noWrap
        , boxShadow4 (px 0) (px 4) (px 16) (rgba 30 41 59 0.2)
        , Css.property "z-index" "99"
        ]

hourBlock : Css.Style
hourBlock =
    Css.batch
        [ backgroundColor (hex "353f4e")
        , borderRadius (px 8)
        , boxShadow4 (px 0) (px 2) (px 12) (rgba 17 24 39 0.15)
        , marginBottom (px 16)
        , marginRight (px 8)
        , width (px 92)
        , height (px 92)
        , displayFlex
        , flexDirection column
        , alignItems center
        , justifyContent spaceBetween
        , padding (px 0)
        ]
