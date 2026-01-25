module DailyQuotaCard exposing (dailyQuotaCard, Device)

import Html exposing (Html, div, span, text)
import Html.Attributes exposing (style)

-- Copying the Device alias so the component is reusable; in a large app, use a shared Types module.
type alias Device =
    { quota : Int
    , dailyActiveMinutes : Int
    }

dailyQuotaCard : Device -> Html msg
dailyQuotaCard device =
    let
        used = device.dailyActiveMinutes
        quota = device.quota
        usagePct : Float
        usagePct =
            if quota <= 0 then 0 else (toFloat used / toFloat quota) * 100
        isOver = quota > 0 && used > quota
        formatTime mins =
            let
                h = mins // 60
                m = modBy 60 mins
            in
            (String.fromInt h) ++ "h " ++ (String.fromInt m) ++ "m"
        quotaLabel = if quota <= 0 then "No quota set" else formatTime used ++ " used"
        rightLabel = if quota <= 0 then "No quota" else formatTime quota ++ " limit"
        barColor = if isOver then "linear-gradient(to right, #ef4444 0%, #dc2626 100%)" else "linear-gradient(to right, #6366f1 0%, #8b5cf6 100%)"
        statusText = if quota <= 0 then "WITHIN LIMITS" else if isOver then "QUOTA EXCEEDED" else "WITHIN LIMITS"
        statusBg = if isOver then "#fef2f2" else if quota <= 0 then "#f8fafc" else "#ecfdf5"
        statusColor = if isOver then "#dc2626" else if quota <= 0 then "#94a3b8" else "#059669"
        statusBorder = if isOver then "#fee2e2" else if quota <= 0 then "#e2e8f0" else "#d1fae5"
    in
    div [ style "background" "#fff"
        , style "border-radius" "20px"
        , style "box-shadow" "0 2px 16px rgba(17,24,39,0.07)"
        , style "padding" "24px"
        , style "border" "1px solid #f1f5f9"
        , style "display" "flex"
        , style "flex-direction" "column"
        ]
        [ span [ style "font-size" "17px", style "font-weight" "600", style "color" "#1e293b" ] [ text "Daily Quota Usage" ]
        , span [ style "color" "#64748b", style "font-size" "13px", style "margin-bottom" "12px" ] [ text "Resets at 00:00 UTC" ]
        , div [ style "display" "flex", style "align-items" "center", style "margin-top" "10px", style "margin-bottom" "24px" ]
            [ span [ style "padding" "4px 14px", style "border-radius" "9999px", style "font-size" "12px", style "font-weight" "700", style "background" statusBg, style "color" statusColor, style "border" ("1.5px solid " ++ statusBorder), style "letter-spacing" "0.02em" ]
                [ text statusText ]
            ]
        , div [ style "display" "flex", style "justify-content" "space-between", style "font-size" "14px", style "font-weight" "500", style "margin-bottom" "4px", style "color" "#64748b" ]
            [ span [] [ text quotaLabel ] 
            , span [ style "color" "#94a3b8" ] [ text rightLabel ] ]
        , div [ style "position" "relative", style "width" "100%", style "height" "16px", style "background" "#f1f5f9", style "border-radius" "8px", style "overflow" "hidden" ]
            [ div [ style "height" "100%", style "width" (if quota <= 0 then "0%" else String.fromFloat (min 100 usagePct) ++ "%"), style "background" barColor, style "transition" "width 1s cubic-bezier(.4,0,.2,1)", style "border-radius" "8px" ] []
            , div [ style "position" "absolute", style "top" "100%", style "left" "50%", style "transform" "translate(-50%, -2px)", style "width" "2px", style "height" "8px", style "background" "#cbd5e1", style "border-radius" "1px" ] []
            , div [ style "position" "absolute", style "top" "100%", style "left" "75%", style "transform" "translate(-50%, -2px)", style "width" "2px", style "height" "8px", style "background" "#cbd5e1", style "border-radius" "1px" ] []
            ]
        ]
