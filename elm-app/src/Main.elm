module Main exposing (main)

import Browser
import Html exposing (Html, div, h2, ul, li, text, span, div, Html, Attribute)
import Html.Attributes exposing (style, class)
import Html.Events exposing (onMouseEnter, onMouseLeave)
import Time
import Task
import Http
import Json.Decode as Decode exposing (Decoder)
import List.Extra exposing (getAt)

import TimelineStyles exposing (intervalBlock, legendActive, legendInactive, tooltip, hourBlock)

import Bulma.Layout exposing (container)
import Bulma.Elements exposing (title)
import Bulma.CDN exposing (stylesheet)

import DailyQuotaCard
import ActiveTimeCard

intervalMinutes : Int
intervalMinutes = 15

intervalsPerDay : Int
intervalsPerDay = 24 * 60 // intervalMinutes

mark : List Bool -> Int -> List Bool
mark used i =
    if i < 0 || i >= intervalsPerDay then
        used
    else
        List.indexedMap (\j b -> if j == i then True else b) used

applyMarkRange : (Int, Int) -> List Bool -> List Bool
applyMarkRange (fromIdx, count) used =
    List.foldl (\d acc -> mark acc (fromIdx + d)) used (List.range 0 (count - 1))

-- MODEL

type alias Device =
    { mac : String
    , name : String
    , dailyActiveMinutes : Int
    , activeSlots : List String
    , quota : Int
    }

type alias Status =
    { devicesChecked : Int
    , usersFetched : Int
    , devices : List Device
    }

type alias Model =
    { status : Maybe Status
    , now : Time.Posix
    , zone : Time.Zone
    , hoveredTimelineBox : Maybe (String, Int) -- (DeviceName, TimelineIdx)
    , hoveredHourCard : Maybe (String, Int) -- (DeviceName, hourIdx)
    }


-- INIT & HTTP

url : String
url =
    "http://localhost:8080/status"

init : () -> ( Model, Cmd Msg )
init _ =
    ( { status = Nothing, now = Time.millisToPosix 0, zone = Time.utc, hoveredTimelineBox = Nothing, hoveredHourCard = Nothing }, Cmd.batch [ fetchStatus, Time.now |> Task.perform NowIs, Time.here |> Task.perform GotZone ] )

fetchStatus : Cmd Msg
fetchStatus =
    Http.get
        { url = url
        , expect = Http.expectJson GotStatus statusDecoder
        }

type Msg
    = GotStatus (Result Http.Error Status)
    | NowIs Time.Posix
    | GotZone Time.Zone
    | TimelineBoxHovered String Int Bool -- deviceName, interval index, hover state
    | HourCardHovered String Int Bool -- deviceName, hourIdx, hover state

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotStatus result ->
            case result of
                Ok status ->
                    ( { model | status = Just status }, Cmd.none )
                Err err ->
                    ( { model | status = Nothing }, Cmd.none )

        NowIs newNow ->
            ( { model | now = newNow }, Cmd.none )

        GotZone z ->
            ( { model | zone = z }, Cmd.none )

        TimelineBoxHovered dev idx hover ->
            let newVal = if hover then Just (dev, idx) else Nothing
            in ({ model | hoveredTimelineBox = newVal }, Cmd.none)

        HourCardHovered dev hour hover ->
            let newVal = if hover then Just (dev, hour) else Nothing
            in ({ model | hoveredHourCard = newVal }, Cmd.none)

httpErrorToString : Http.Error -> String
httpErrorToString err =
    case err of
        Http.BadUrl msg -> "Bad URL: " ++ msg
        Http.Timeout -> "Request timed out"
        Http.NetworkError -> "Network error"
        Http.BadStatus status -> "Bad status: " ++ String.fromInt status
        Http.BadBody msg -> "Bad body: " ++ msg

-- DECODERS

deviceDecoder : Decoder Device
deviceDecoder =
    Decode.map5 Device
        (Decode.field "mac" Decode.string)
        (Decode.field "name" Decode.string)
        (Decode.field "daily_active_minutes" Decode.int)
        (Decode.field "active" (Decode.list Decode.string))
        (Decode.field "quota" Decode.int)

statusDecoder : Decoder Status
statusDecoder =
    Decode.map3 Status
        (Decode.field "DevicesChecked" Decode.int)
        (Decode.field "UsersFetched" Decode.int)
        (Decode.field "devices" (Decode.list deviceDecoder))

-- VIEW

minuteMs : Int
minuteMs = 60 * 1000

subscriptions : Model -> Sub Msg
subscriptions _ =
    Time.every (toFloat minuteMs) NowIs

currentInterval : Time.Zone -> Time.Posix -> Int
currentInterval zone now =
    let
        hours = Time.toHour zone now
        mins = Time.toMinute zone now
        totalMinutes = hours * 60 + mins
    in
        totalMinutes // intervalMinutes

view : Model -> Html Msg
view model =
    container [] (
        stylesheet ::
        case model.status of
            Nothing ->
                [ h2 [ Html.Attributes.class "title is-2" ] [ text "Loading…" ] ]
            Just status ->
                [ devicesListView status.devices (currentInterval model.zone model.now) model.hoveredTimelineBox model.hoveredHourCard identity
                ]
    )


devicesListView devices currSeg hoveredTimelineBox hoveredHourCard liftMsg =
    ul [] (List.map (\dev -> deviceTimelineView dev currSeg hoveredTimelineBox hoveredHourCard liftMsg) devices)

hourMarkerView : Int -> Html msg
hourMarkerView idx =
    if modBy 4 (idx + 1) == 0 then
        let
            hour = String.fromInt ((idx + 1) // 4)
        in
        div [ style "display" "flex", style "align-items" "center", style "width" "2px", style "height" "32px", style "margin-right" "6px" ]
            [ div [ style "width" "2px", style "height" "32px", style "background-color" "#333", style "margin-right" "4px" ] []
            , span [ style "color" "#aaa", style "font-size" "13px", style "padding-left" "4px" ] [ text hour ]
            ]
    else
        div [ style "width" "8px" ] []

hourContainerView : String -> Int -> List (Int, Bool, Bool) -> Int -> Maybe (String, Int) -> Maybe (String, Int) -> (Msg -> msg) -> Html msg
hourContainerView deviceName h timeline currSeg hoveredTimelineBox hoveredHourCard liftMsg =
    let
        startIdx = h * 4
        segmentViews =
            List.map (\i ->
                let
                    (idx, isA, isOverQ) = List.Extra.getAt i timeline |> Maybe.withDefault (i, False, False)
                    isCurrent = (i == currSeg)
                    isHovered = hoveredTimelineBox == Just (deviceName, i)
                in
                timelineBoxView deviceName i isA isCurrent isHovered isOverQ liftMsg
            ) (List.range startIdx (startIdx + 3))
        isHourHovered = hoveredHourCard == Just (deviceName, h)
        hourBoxBg = if isHourHovered then "rgba(248,250,252,0.92)" else "rgba(248,250,252,0.80)" -- Tailwind bg-slate-50/50 hover:bg-slate-50/90
        hourBoxBorder = if isHourHovered then "1px solid #e5e7eb" else "1px solid rgba(241,245,249,0.8)" --  border-slate-200 or border-slate-100/50
    in
    div
        ([ style "background-color" hourBoxBg
         , style "border-radius" "12px"
         , style "box-shadow" "0 2px 12px rgba(17,24,39,0.09)"
         , style "border" hourBoxBorder
         , style "display" "flex"
         , style "flex-direction" "column"
         , style "align-items" "center"
         , style "justify-content" "space-between"
         , style "margin" "0"
         , style "padding" "12px"
         , style "transition" "border-color 0.21s cubic-bezier(.25,.8,.50,1), background-color 0.21s cubic-bezier(.25,.8,.50,1)"
         , onMouseEnter (liftMsg (HourCardHovered deviceName h True))
         , onMouseLeave (liftMsg (HourCardHovered deviceName h False))
        ]
        )
        [ span [ style "color" "#a1a1aa" -- slate-400
               , style "font-size" "12px"
               , style "font-weight" "600"
               , style "margin-bottom" "8px"
               , style "align-self" "flex-start"
               ] [ text ((if h < 10 then "0" else "") ++ String.fromInt h ++ ":00") ]
        , div [ style "display" "flex", style "gap" "4px", style "align-items" "end", style "height" "32px" ] segmentViews
        , div [ style "display" "flex", style "width" "100%", style "justify-content" "space-between", style "margin-top" "4px", style "padding-left" "1px", style "padding-right" "1px" ]
             [ span [ style "font-size" "8px", style "color" "#a1a1aa" ] [ text "00" ]
             , span [ style "font-size" "8px", style "color" "#a1a1aa" ] [ text "60" ]
             ]
        ]

deviceTimelineView : Device -> Int -> Maybe (String, Int) -> Maybe (String, Int) -> (Msg -> msg) -> Html msg
deviceTimelineView device currSeg hoveredTimelineBox hoveredHourCard liftMsg =
    let
        timeline = makeTimeline device.activeSlots device.quota
        name = device.name
        isOverQuota = device.quota > 0 && device.dailyActiveMinutes > device.quota
    in
    div [ Html.Attributes.class "box" ]
        [ h2 [ Html.Attributes.class "title is-4" ] [ text device.name ]
        , div [ style "display" "grid", style "grid-template-columns" "2fr 1fr", style "gap" "24px", style "margin-bottom" "24px" ]
            [ DailyQuotaCard.dailyQuotaCard { quota = device.quota, dailyActiveMinutes = device.dailyActiveMinutes }
            , ActiveTimeCard.activeTimeCard { quota = device.quota, dailyActiveMinutes = device.dailyActiveMinutes }
            ]
        , div [ style "display" "flex", style "flex-wrap" "wrap", style "gap" "12px", style "justify-content" "flex-start", style "margin" "16px 0 0 0" ]
            (List.map (\h -> hourContainerView name h timeline currSeg hoveredTimelineBox hoveredHourCard liftMsg) (List.range 6 21))
        ]

timelineBoxView : String -> Int -> Bool -> Bool -> Bool -> Bool -> (Msg -> msg) -> Html msg
timelineBoxView deviceName idx isActive isCurrent isHovered isOverQuota liftMsg =
    let
        (tStart, tEnd) = intervalToTimes idx
        statusStr = if isActive then "Active" else "Inactive"
        tooltipContent = tStart ++ "-" ++ tEnd ++ " (" ++ statusStr ++ ")"
        baseStyles =
            [ style "display" "inline-block"
            , style "vertical-align" "bottom"
            , style "margin-right" "3px"
            , style "position" "relative"
            ]
            ++ (if isCurrent then
                    [ style "border" "1px solid #242424" ]
               else
                    [])
            ++ [ style "width" "18px"
               , style "height" "32px"
               , style "border-radius" "4px"
               , style "background-color" (
                    if isActive then
                        if isOverQuota then (if isHovered then "#ef4444" else "#dc2626")
                        else if isHovered then "#34d399" else "#10b981"
                    else
                        if isHovered then "#d1d5db" else "#e5e7eb"
                 )
               , style "transition" "transform 0.17s cubic-bezier(.25,.8,.50,1), box-shadow 0.17s cubic-bezier(.25,.8,.50,1), background-color 0.12s cubic-bezier(.25,.8,.50,1)"
               , style "transform" (
                     if isHovered then
                        if isActive then "scaleY(1.12)" else "scaleY(0.90)"
                     else if isActive then "scaleY(0.90)" else "scaleY(0.75)"
                 )
               , style "box-shadow" (
                    if isActive then
                        if isHovered then "0 0 14px 0 rgba(16,185,129,0.35)"
                        else "0 0 8px 0 rgba(16,185,129,0.19)"
                    else
                        if isHovered then "0 0 4px 0 #bbbbbb"
                        else "none"
                 )
               , style "cursor" "pointer"
               ]

    in
    span
        ([ onMouseEnter (liftMsg (TimelineBoxHovered deviceName idx True))
         , onMouseLeave (liftMsg (TimelineBoxHovered deviceName idx False))
        ]
        ++ baseStyles
        )
        [ if isHovered then
                div [ style "position" "absolute"
                    , style "bottom" "110%"
                    , style "left" "50%"
                    , style "transform" "translateX(-50%)"
                    , style "background-color" "#111827"
                    , style "color" "#fff"
                    , style "font-size" "12px"
                    , style "padding" "6px 12px"
                    , style "border-radius" "4px"
                    , style "white-space" "nowrap"
                    , style "box-shadow" "0 4px 16px rgba(30,41,59,0.2)"
                    , style "z-index" "99"
                    ]
                    [ text tooltipContent ]
           else text ""
        ]

intervalToTimes : Int -> (String, String)
intervalToTimes idx =
    let
        minutes = idx * intervalMinutes
        hour = minutes // 60
        minute = modBy 60 minutes
        nextMinutes = minutes + intervalMinutes
        nextHour = nextMinutes // 60
        nextMinute = modBy 60 nextMinutes
        pad n = if n < 10 then "0" ++ String.fromInt n else String.fromInt n
        tStart = pad hour ++ ":" ++ pad minute
        tEnd = pad nextHour ++ ":" ++ pad nextMinute
    in
    (tStart, tEnd)

makeTimeline : List String -> Int -> List (Int, Bool, Bool)
makeTimeline activeSlots quota =
    let
        all = List.repeat intervalsPerDay False
        marks = List.filterMap parseSlot activeSlots
        activeArray = List.foldl applyMarkRange all marks
        -- Each slot: (interval_idx, isActive, isOverQuotaActive)
        foldActive (idx, isActive) (accum, result) =
            if not isActive then
                (accum, (idx, False, False) :: result)
            else
                let
                    isOver = quota > 0 && accum >= quota
                    nextAccum = accum + intervalMinutes
                in
                    (nextAccum, (idx, True, isOver) :: result)
        (_, withOverQuotaReversed) =
            List.foldl foldActive (0, []) (List.indexedMap (\i b -> (i, b)) activeArray)
    in
        List.reverse withOverQuotaReversed


parseSlot : String -> Maybe (Int, Int)
parseSlot slotString =
    let
        timeAndDur = String.split "/" slotString
        toIdx time =
            case String.split ":" time of
                hourStr :: minAndZone :: _ ->
                    case (String.toInt hourStr, String.left 2 minAndZone |> String.toInt) of
                        (Just h, Just m) ->
                            (h * 60 + m) // intervalMinutes
                        _ -> 0
                _ -> 0
        parseDuration s =
            let
                get = String.dropLeft 2 s
                (dh, rest) =
                    if String.startsWith "PT" s && String.contains "H" get then
                        case String.split "H" get of
                            hour :: r :: _ -> (String.toInt hour, r)
                            hour :: _ -> (String.toInt hour, "")
                            _ -> (Nothing, "")
                    else (Nothing, get)
                minPart =
                    if String.contains "M" rest then
                        String.left (Maybe.withDefault 0 (String.indexes "M" rest |> List.head)) rest
                            |> String.toInt
                    else
                        Nothing
            in
            case (dh, minPart) of
                (Just h, Just m) -> h * 60 + m
                (Just h, Nothing) -> h * 60
                (Nothing, Just m) -> m
                _ -> 0
    in
    case timeAndDur of
        [start, dur] ->
            let
                baseTime = String.left 5 start -- "09:45"
                idx = toIdx baseTime
                mins = parseDuration dur
                count = (mins + intervalMinutes - 1) // intervalMinutes
            in
            Just (idx, count)
        _ -> Nothing

main : Program () Model Msg
main =
    Browser.element
        { init = init
        , update = update
        , view = view
        , subscriptions = \_ -> Sub.none
        }
