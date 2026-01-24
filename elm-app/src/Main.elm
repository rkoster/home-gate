module Main exposing (main)

import Browser
import Html exposing (Html, div, h2, ul, li, text, span)
import Html.Attributes exposing (style)
import Http
import Json.Decode as Decode exposing (Decoder)
import List.Extra exposing (getAt)

import Bulma.Layout exposing (container)
import Bulma.Elements exposing (title)
import Bulma.CDN exposing (stylesheet)

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
    }

type alias Status =
    { devicesChecked : Int
    , usersFetched : Int
    , devices : List Device
    }

type Model
    = Loading
    | Failure String
    | Success Status

-- INIT & HTTP

url : String
url =
    "http://localhost:8080/status"

init : () -> ( Model, Cmd Msg )
init _ =
    ( Loading, fetchStatus )

fetchStatus : Cmd Msg
fetchStatus =
    Http.get
        { url = url
        , expect = Http.expectJson GotStatus statusDecoder
        }

type Msg
    = GotStatus (Result Http.Error Status)

update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotStatus result ->
            case result of
                Ok status ->
                    ( Success status, Cmd.none )
                Err err ->
                    ( Failure (httpErrorToString err), Cmd.none )

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
    Decode.map4 Device
        (Decode.field "mac" Decode.string)
        (Decode.field "name" Decode.string)
        (Decode.field "daily_active_minutes" Decode.int)
        (Decode.field "active" (Decode.list Decode.string))

statusDecoder : Decoder Status
statusDecoder =
    Decode.map3 Status
        (Decode.field "DevicesChecked" Decode.int)
        (Decode.field "UsersFetched" Decode.int)
        (Decode.field "devices" (Decode.list deviceDecoder))

-- VIEW

view : Model -> Html Msg
view model =
    case model of
        Loading ->
            container []
                [ stylesheet
                , h2 [ Html.Attributes.class "title is-2" ] [ text "Loading…" ]
                ]
        Failure err ->
            container []
                [ stylesheet
                , h2 [ Html.Attributes.class "title is-2 has-text-danger" ] [ text ("Error: " ++ err) ]
                ]
        Success status ->
            container []
                [ stylesheet
                , h2 [ Html.Attributes.class "title is-2" ] [ text "Device Timelines" ]
                , devicesListView status.devices
                ]

devicesListView : List Device -> Html Msg
devicesListView devices =
    ul [] (List.map deviceTimelineView devices)

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

hourContainerView : Int -> List (Int, Bool) -> Html msg
hourContainerView h timeline =
    let
        startIdx = h * 4
        segmentViews =
            List.map (\i ->
                let
                    (_, isA) = List.Extra.getAt i timeline |> Maybe.withDefault (i, False)
                in
                timelineBoxView i isA
            ) (List.range startIdx (startIdx + 3))
    in
    div
        [ style "display" "flex"
        , style "flex-direction" "column"
        , style "align-items" "center"
        , style "border-left" "2px solid #bbb"
        , style "width" "40px"
        , style "padding-left" "0px"
        ]
        [ span [ style "color" "#aaa", style "font-size" "11px", style "margin-bottom" "2px", style "align-self" "flex-start", style "margin-left" "2px" ] [ text (String.fromInt h) ]
        , div [ style "display" "flex" ] segmentViews
        ]

deviceTimelineView : Device -> Html Msg
deviceTimelineView device =
    let
        timeline = makeTimeline device.activeSlots
    in
    div [ Html.Attributes.class "box" ]
        [ h2 [ Html.Attributes.class "title is-4" ] [ text device.name ]
        , div [] [ text ("Total minutes today: " ++ String.fromInt device.dailyActiveMinutes) ]
    , div [ style "display" "flex", style "margin" "6px 0" ]
        (List.map (\h -> hourContainerView h timeline) (List.range 0 23))
    ]

timelineBoxView : Int -> Bool -> Html msg
timelineBoxView idx isActive =
    let
        (tStart, tEnd) = intervalToTimes idx
        statusStr = if isActive then "Active" else "Inactive"
        tooltip = tStart ++ "-" ++ tEnd ++ " (" ++ statusStr ++ ")"
    in
    span
        [ style "display" "inline-block"
        , style "width" "8px"
        , style "height" "16px"
        , style "margin-right" "1px"
        , style "background-color" (if isActive then "#30c750" else "#bbb")
        , Html.Attributes.title tooltip
        ]
        []

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

makeTimeline : List String -> List (Int, Bool)
makeTimeline activeSlots =
    let
        all = List.repeat intervalsPerDay False
        marks = List.filterMap parseSlot activeSlots
        activeArray = List.foldl applyMarkRange all marks
    in
    List.indexedMap (\i b -> (i, b)) activeArray


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
