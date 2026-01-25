module ActiveTimeCard exposing (activeTimeCard, Device)

import Html exposing (Html, div, span, text)
import Html.Attributes exposing (style)

-- Copy Device alias for self-containment; for production, use a shared module.
type alias Device =
    { quota : Int
    , dailyActiveMinutes : Int
    }

activeTimeCard : Device -> Html msg
activeTimeCard device =
    let
        h = device.dailyActiveMinutes // 60
        m = modBy 60 device.dailyActiveMinutes
        pretty = (String.fromInt h) ++ "h " ++ (String.fromInt m) ++ "m"
    in
    div [ style "background" "#fff"
        , style "border-radius" "20px"
        , style "box-shadow" "0 2px 16px rgba(17,24,39,0.07)"
        , style "padding" "24px"
        , style "border" "1px solid #f1f5f9"
        , style "display" "flex"
        , style "flex-direction" "column"
        , style "justify-content" "center"
        , style "align-items" "flex-start"
        ]
        [ div [ style "padding" "12px", style "background" "#ecfdf5", style "border-radius" "16px", style "margin-bottom" "10px", style "display" "flex", style "align-items" "center", style "justify-content" "center" ]
            [ span [ style "color" "#059669", style "font-size" "24px", style "font-weight" "bold" ] [ text "✓" ] ]
        , span [ style "font-size" "11px", style "text-transform" "uppercase", style "color" "#64748b", style "font-weight" "600", style "margin-bottom" "2px", style "letter-spacing" "0.08em" ] [ text "Active Time" ]
        , span [ style "font-size" "21px", style "font-weight" "bold", style "color" "#1e293b" ] [ text pretty ]
        ]
